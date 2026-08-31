"""The autonomous supervisor loop.

Drives the multi-phase build by:
  1) reading the master requirements + phase state,
  2) delegating the next task to the appropriate role via the opencode bridge,
  3) verifying (build/test) that the change is green,
  4) healing on failure (bounded retries), otherwise escalating,
  5) advancing phase to the next when the gate is met,
  6) pausing at user gates (phase boundary + prod deploy).

Usage:
  python orchestrator.py                 # run from current state
  python orchestrator.py --phase 0       # force a phase
  python orchestrator.py --only P0-1     # run one task id
  python orchestrator.py --dry-run       # print next actions, no delegation
"""

from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
import time

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from crew import state as st  # noqa: E402
from crew.roles import render_task  # noqa: E402
from tools.opencode_runner import OpencodeRunnerTool  # noqa: E402

MAX_HEAL = 6  # heal iterations before escalating a single task

# Role(s) used per phase for planning. Analysts/PM refine requirements before dev.
PLANNERS = {0: "analyst", 1: "product_manager", 2: "product_manager", 3: "analyst", 4: "analyst"}
# Role used to implement tasks of a phase.
DEV_ROLE = {
    0: "frontend_engineer",  # baseline touches all surfaces; frontend is safest common
    1: "backend_engineer",
    2: "backend_engineer",
    3: "sre",  # Phase 3 is architecture-heavy (L4 LB, registry, scale)
    4: "backend_engineer",
}
VERIFY_ROLE = "qa_engineer"
REVIEW_ROLE = "reviewer"
TEACHER_ROLE = "teacher"


def log(msg: str) -> None:
    print(f"[{time.strftime('%H:%M:%S')}] {msg}", flush=True)


def run_bridge(prompt: str, suffix: str = "general", job: str = "") -> dict:
    """Call opencode via the bridge; returns the structured result dict."""
    raw = OpencodeRunnerTool()._run(prompt, agent_suffix=suffix, job_id=job)
    try:
        return json.loads(raw)
    except json.JSONDecodeError:
        return {"ok": False, "summary": raw, "stderr": raw, "exit_code": 1}


REPO_ROOT = os.path.dirname(os.path.abspath(__file__))


def verify(state: dict, task_id: str) -> dict:
    """Run the phase's green gate deterministically and report PASS/FAIL.

    Executes the canonical layer commands itself (mirroring RUN_GUIDE.md's green
    gate) and judges purely on exit codes. The previous version delegated to a QA
    agent that was (a) routed to a subagent opencode rejects (falls back to the
    default agent) and (b) whose final verdict was silently discarded because its
    text is emitted at ``part.text``, not ``message`` -- so a healthy green build
    was reported as "verification failed" and the heal loop spun.
    """
    checks = {
        0: [
            "cd backend && go build ./... && go test ./...",
            "cd frontend && npm test",
            "cd mobile && npm test",
        ],
        1: [
            "cd backend && go test ./...",
            "cd frontend && npm test",
            "cd frontend && npm run build",
        ],
        2: ["cd backend && go test ./...", "cd frontend && npm test", "cd mobile && npm test"],
        3: ["cd backend && go test ./...", "cd e2e && npx playwright test"],
        4: ["cd backend && go test ./...", "cd frontend && npm run build"],
    }
    cmd_list = checks.get(st.current_phase(state)["id"], ["cd backend && go test ./..."])
    stdout_lines: list[str] = []
    stderr_lines: list[str] = []
    failures: list[str] = []
    for cmd in cmd_list:
        code, out, err = _run_command(cmd)
        stdout_lines.append(f"$ {cmd}\n{out.strip()}")
        if err.strip():
            stderr_lines.append(f"$ {cmd}\n{err.strip()}")
        if code != 0:
            failures.append(f"$ {cmd}\n{err.strip() or out.strip()}")

    if failures:
        ok = False
        summary = "FAIL\n" + "\n".join(failures)
    else:
        ok = True
        summary = "PASS\n" + "\n".join(stdout_lines)

    return {
        "job_id": f"verify_{task_id}",
        "ok": ok,
        "error": None,
        "stdout": "\n".join(stdout_lines)[-8000:],
        "stderr": "\n".join(stderr_lines)[-8000:],
        "exit_code": 0 if ok else 1,
        "elapsed": 0.0,
        "summary": summary[-2500:],
        "files_changed": [],
    }


def _run_command(cmd: str) -> tuple[int, str, str]:
    """Run one canonical gate command and return (exit_code, stdout, stderr)."""
    import subprocess

    try:
        proc = subprocess.run(
            cmd,
            shell=True,
            capture_output=True,
            text=True,
            encoding="utf-8",
            errors="replace",
            cwd=REPO_ROOT,
            timeout=1200,
        )
        return proc.returncode, proc.stdout or "", proc.stderr or ""
    except subprocess.TimeoutExpired:
        return -1, "", f"timed out after 1200s: {cmd}"


def phase_plans(state: dict) -> dict | None:
    """Ask the product manager/analyst to re-plan the current phase before dev runs."""
    if st.phase_done(state):
        return None
    prompt = (
        "You are the product manager. Read REQUIREMENTS_MASTER.md (Section for the active "
        "phase), WORKING_SET.md, and skim the codebase to confirm the backlog is accurate. "
        "List the CURRENT phase's pending tasks in priority order and flag any that look "
        "already done or are actually out of scope. Return a concise prioritized list and, "
        "for each task, 2-3 concrete files/endpoints to touch."
    )
    result = run_bridge(prompt, suffix=PLANNERS.get(st.current_phase(state)["id"], "admin"),
                        job="plan")
    return result


def teacher_review(state: dict, task_id: str, summary: str) -> dict:
    """For learning-related tasks, get the bilingual teacher sign-off."""
    prompt = (
        "You are the bilingual ES/EN language teacher reviewer. The task was completed with "
        f"this summary:\n{summary}\n\n"
        "Evaluate whether the language-learning outputs (translations, grammar feedback, CEFR "
        "labelling, lesson/vocab content, scenario scripts) are pedagogically sound and "
        "linguistically correct. Return PASS or NEEDS-CHANGES with concrete critiques. Be "
        "strict: a learner-facing translation or grammar claim must be correct."
    )
    return run_bridge(prompt, suffix="teacher", job=f"teacher_{task_id}")


def escalate(state: dict, task_id: str, reason: str) -> None:
    path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "ESCALATION.md")
    with open(path, "a", encoding="utf-8") as f:
        f.write(f"\n## Escalation {time.strftime('%Y-%m-%d %H:%M')} — task {task_id}\n{reason}\n")
    log(f"! Escalated task {task_id}. See crew/ESCALATION.md")


def human_gate(state: dict, label: str) -> bool:
    """Pause for the human owner at each phase boundary / prod deploy."""
    print(f"\n{GREEN}=== HUMAN GATE: {label} ==={RESET}", flush=True)
    ans = input("Approve to proceed? (y/N): ").strip().lower()
    return ans in ("y", "yes")


GREEN = "\033[0;32m"
RED = "\033[0;31m"
YELLOW = "\033[0;33m"
RESET = "\033[0m"


def run_task(state: dict, task_id: str, dry_run: bool) -> None:
    phase_id = st.current_phase(state)["id"]
    task = next(t for t in st.current_phase(state)["tasks"] if t["id"] == task_id)
    dev_role = DEV_ROLE.get(phase_id, "backend_engineer")

    prompt = (
        f"You are the {dev_role} role. Implement this task: {task['name']} (id {task_id}).\n"
        "Read REQUIREMENTS_MASTER.md for the exact requirement, then the relevant code, then "
        "make minimal focused changes under WORKING_SET.md's ALLOWED paths. After editing, run "
        "that layer's build/test and report exit code. Give a short summary of what changed."
    )

    if dry_run:
        log(f"[dry-run] would delegate task {task_id} to {dev_role}")
        st.mark_task(state, task_id, "IN_PROGRESS", "dry-run")
        return

    log(f"{YELLOW}> dev task {task_id}: {task['name']} ({dev_role}){RESET}")
    result = run_bridge(prompt, suffix=dev_role, job=f"dev_{task_id}")
    summary = (result.get("summary") or "")[:2000]
    if not result.get("ok"):
        log(f"{YELLOW}    dev finished non-ok, healing...{RESET}")

    # Verify + heal loop
    for attempt in range(1, MAX_HEAL + 1):
        v = verify(state, task_id)
        ok = v.get("ok") and "PASS" in (v.get("summary") or "").upper()
        if ok:
            log(f"{GREEN}OK task {task_id} verified green (attempt {attempt}){RESET}")
            break
        log(f"{YELLOW}    heal {attempt}/{MAX_HEAL} - verification failed, delegating fix...{RESET}")
        if dry_run:
            st.mark_task(state, task_id, "IN_PROGRESS", "verify failed (dry-run)")
            return
        fix_prompt = (
            f"Verification failed for task {task_id} ({task['name']}). QA output:\n"
            f"{(v.get('summary') or v.get('stderr') or '')[:2500]}\n\n"
            "Fix the build/test failure. Read the actual error, correct the code, re-run the "
            "same command until it exits 0. Report the final exit code."
        )
        fix = run_bridge(fix_prompt, suffix=dev_role, job=f"fix_{task_id}_{attempt}")
        if not fix.get("ok"):
            escalate(state, task_id, f"heal {attempt}: fix delegated but non-zero.\n{fix.get('summary','')}")
            return

    else:
        escalate(state, task_id, f"still failing after {MAX_HEAL} heals.")
        st.mark_task(state, task_id, "BLOCKED")
        return

    # Language-learning subset gets teacher review
    if phase_id in (1, 3, 4) and any(k in task["name"].lower() for k in
                                   ("grammar", "word", "learn", "vocab", "scenario", "lesson",
                                    "path", "writing", "translate", "srs", "call", "caption")):
        tr = teacher_review(state, task_id, summary)
        if "NEEDS-CHANGES" in (tr.get("summary") or "").upper():
            log(f"{YELLOW}    teacher requested changes, delegating...{RESET}")
            fix = run_bridge(
                "Teacher review raised: " + (tr.get("summary") or "")[:2000] +
                "\nAddress the pedagogical/linguistic issues and re-run tests.",
                suffix=dev_role, job=f"teacherfix_{task_id}")
            if not fix.get("ok"):
                escalate(state, task_id, "teacher changes not resolved.")

    st.mark_task(state, task_id, "DONE", summary[:500])


def advance(state: dict, dry_run: bool) -> None:
    if st.phase_done(state):
        if dry_run:
            log(f"[dry-run] phase {st.current_phase(state)['id']} gate met — would advance")
            return
        me = st.current_phase(state)["name"]
        if not human_gate(state, f"Phase {st.current_phase(state)['id']}: {me} complete — advance?"):
            log("Held at user gate. Resolve and re-run.")
            return
        st.advance_phase(state)
        st.save(state)
        log(f"-> advanced to Phase {st.current_phase(state)['id']} ({st.current_phase(state)['name']})")
    else:
        log(f"{YELLOW}phase {st.current_phase(state)['id']} still has pending tasks.{RESET}")


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--phase", type=int, default=None)
    ap.add_argument("--only", dest="only", default=None, help="run a single task id")
    ap.add_argument("--dry-run", action="store_true")
    ap.add_argument("--no-plan", action="store_true", help="skip the re-plan step")
    args = ap.parse_args()

    state = st.load()
    if args.phase is not None:
        state["current_phase"] = args.phase
        st.save(state)

    log(f"Autonomous Chorus build — Phase {st.current_phase(state)['id']}: "
        f"{st.current_phase(state)['name']}")

    if not args.dry_run and not args.no_plan:
        plan = phase_plans(state)
        log(f"{YELLOW}plan/{PLANNERS.get(st.current_phase(state)['id'], 'admin')} re-plan:{RESET} "
            f"{(plan.get('summary') or 'n/a')[:1200] if plan else 'n/a'}")

    if args.only:
        if any(t["id"] == args.only for t in st.current_phase(state)["tasks"]):
            run_task(state, args.only, args.dry_run)
            if not args.dry_run:
                st.save(state)
        else:
            log(f"task {args.only} not found in current phase")
        return 0

    # Drain the current phase
    while True:
        task = st.next_task(state)
        if task is None:
            break
        run_task(state, task["id"], args.dry_run)
        # In dry-run, mark the task so we don't loop forever and do not persist.
        if args.dry_run and isinstance(task, dict):
            st.mark_task(state, task["id"], "DONE", "dry-run")

    if not args.dry_run:
        st.save(state)
        advance(state, dry_run=False)
    else:
        log("dry-run complete; state not mutated")
        return 0

    # If a next phase exists and is IN_PROGRESS, hand off for the next invocation
    log(f"Phase {st.current_phase(state)['id']} ready. Run again to continue phase "
        f"{st.current_phase(state)['id']} ({st.current_phase(state)['name']}).")
    return 0


if __name__ == "__main__":
    sys.exit(main())
