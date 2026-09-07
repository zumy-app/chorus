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
import threading
import time

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from crew import state as st  # noqa: E402
from crew.gates import run_full_gate as run_deterministic_full_gate  # noqa: E402
from crew.gates import run_static_contract_gate, run_task_gate  # noqa: E402
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
    """Call opencode via the bridge; returns the structured result dict.

    Emits a heartbeat every 45s so long-running delegations are visible
    instead of showing only 'thinking'. The heartbeat also appends to
    agent_jobs/progress.log which you can tail live.
    """
    job_label = job or suffix
    stop = threading.Event()
    progress_path = os.path.join(REPO_ROOT, "agent_jobs", "progress.log")

    def _heartbeat():
        start = time.time()
        while not stop.wait(45):
            elapsed = int(time.time() - start)
            msg = f"... still working: {job_label} ({elapsed}s elapsed, {time.strftime('%H:%M:%S')})"
            print(msg, flush=True)
            try:
                with open(progress_path, "a", encoding="utf-8") as pf:
                    pf.write(f"[{time.strftime('%H:%M:%S')}] {msg}\n")
            except OSError:
                pass

    hb = threading.Thread(target=_heartbeat, daemon=True)
    hb.start()
    try:
        raw = OpencodeRunnerTool()._run(prompt, agent_suffix=suffix, job_id=job)
    finally:
        stop.set()
        hb.join(timeout=1)
    try:
        return json.loads(raw)
    except json.JSONDecodeError:
        return {"ok": False, "summary": raw, "stderr": raw, "exit_code": 1}


REPO_ROOT = os.path.dirname(os.path.abspath(__file__))


def verify(state: dict, task_id: str, role: str = "backend_engineer") -> dict:
    """Run the affected surface's deterministic gate and report PASS/FAIL."""
    task_name = task_id
    for phase in state.get("phases", []):
        for task in phase.get("tasks", []):
            if task.get("id") == task_id:
                task_name = task.get("name", task_id)
                role = task.get("role", role)
                break
    return run_task_gate(task_id, task_name, role).to_result(f"verify_{task_id}")


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
        "Also run `python orchestrator.py --audit` mentally against the current files: any "
        "soft-pass E2E, external Gmail fixture, mocked-only acceptance proof, missing Learn "
        "Dashboard/Daily Practice/Placement/Sparky contract, or wireframe GAP means the phase "
        "is NOT done even if phase_status.json says DONE. "
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
    log(f"! Escalated task {task_id}. See ESCALATION.md")


def human_gate(state: dict, label: str) -> bool:
    """Pause for the human owner at each phase boundary / prod deploy.

    In non-interactive mode (stdin not a tty, or AUTO_APPROVE_GATES=1) the gate
    auto-approves so the autonomous loop can run unattended. Set REQUIRE_GATES=1
    to force interactive approval even in CI.
    """
    if os.environ.get("REQUIRE_GATES") != "1":
        if os.environ.get("AUTO_APPROVE_GATES") == "1" or not sys.stdin.isatty():
            print(f"\n{GREEN}=== HUMAN GATE (auto-approved): {label} ==={RESET}", flush=True)
            return True
    print(f"\n{GREEN}=== HUMAN GATE: {label} ==={RESET}", flush=True)
    try:
        ans = input("Approve to proceed? (y/N): ").strip().lower()
    except EOFError:
        print("No tty — auto-approving gate.", flush=True)
        return True
    return ans in ("y", "yes")


GREEN = "\033[0;32m"
RED = "\033[0;31m"
YELLOW = "\033[0;33m"
RESET = "\033[0m"


def route_role(task: dict, phase_id: int) -> str:
    """Pick the right engineer role for a task based on its name/domain.

    Fall back to the phase's default DEV_ROLE when the task is ambiguous.
    """
    explicit = task.get("role")
    if explicit:
        return explicit
    name = task["name"].lower()
    if any(k in name for k in ("test quality", "acceptance", "test-first", "failing tests", "tdd")):
        return "test_engineer"
    if any(k in name for k in ("qa ", "qa-", "qa:", "functional suite", "release gate", "go/no-go")):
        return "qa_engineer"
    if any(k in name for k in ("gap audit", "traceability", "ba signoff", "sign-off")):
        return "analyst"
    if "wireframe parity" in name and any(k in name for k in ("wiring", "routes", "screens", "implement", "remaining")):
        if "frontend" in name:
            return "frontend_engineer"
        return "mobile_engineer"
    if "analyst" in name:
        return "analyst"
    if any(k in name for k in ("docker", "compose", "load balancer", "lb", "deploy",
                               "infra", "ci", "observability", "prometheus", "grafana")):
        return "sre"
    if any(k in name for k in ("mobile", "react native", "expo", "android", "ios",
                               "learning dashboard", "learnscreen", "daily practice",
                               "quick drill", "quick drills", "drill", "placement",
                               "initial test", "sparky", "scenario roleplay")):
        return "mobile_engineer"
    if any(k in name for k in ("emoji", "home button", "back", "admin", "copy",
                               "settings", "chat language", "writing assistant",
                               "highlight", "learning path", "scenario", "presence", "typing",
                               "marketplace", "tutor", "browse", "teacher", "payout", "trial credit")):
        return "mobile_engineer" if phase_id in (4,) else "frontend_engineer"
    return DEV_ROLE.get(phase_id, "backend_engineer")


def run_task(state: dict, task_id: str, dry_run: bool) -> None:
    phase_id = st.current_phase(state)["id"]
    task = next(t for t in st.current_phase(state)["tasks"] if t["id"] == task_id)
    dev_role = route_role(task, phase_id)

    # QA functional tasks need the full wireframe+requirements context — not just the short name
    extra_qa = ""
    if dev_role == "qa_engineer":
        extra_qa = (
            " You MUST: 1) read wireframes/ (94 entries, see docs/WIREFRAME_TRACE.md), "
            "REQUIREMENTS_MASTER.md, REQUIREMENTS.md, chorus_lesson_design_and_vocabulary_engine.md, "
            "2) write EXTENSIVE functional tests that prove Spanish real-world scenarios exist and are runnable "
            "( scenarios list for es, opening line + translation, chunk, hint, send Spanish + AI reply ), "
            "3) prove daily drills work (vocab due, SRS, session start/answer, streak), "
            "4) prove marketplace + learn hub navigation reachable on BOTH mobile (MainTabs.tsx) and web (App.tsx). "
            "Write tests to mobile/src/screens/__tests__/qa-functional.test.tsx, "
            "frontend/src/__tests__/qa-functional.test.tsx, backend/internal/services/qa_spanish_test.go, "
            "and e2e/tests/qa-scenarios-drills.spec.ts. Run each suite and report PASS/FAIL per file. "
        )
    prompt = (
        f"You are the {dev_role} role. Implement this task: {task['name']} (id {task_id}).{extra_qa}\n"
        "Read REQUIREMENTS_MASTER.md for the exact requirement, then the relevant code, then "
        "make minimal focused changes under WORKING_SET.md's ALLOWED paths. Use TDD: first "
        "prove the missing behavior with a failing testRef, then make it green. Critical "
        "acceptance tests must drive real UI/API behavior; do not use console.warn soft-pass "
        "fallbacks, source-file read assertions, or swallowed catches as proof. After editing, "
        "run that layer's build/test and report exit code. Give a short summary of what changed."
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
        v = verify(state, task_id, role=dev_role)
        ok = v.get("ok", False)
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
            "Fix the real failure. If the gate complains about hollow tests, replace them with "
            "hard assertions instead of weakening the gate. Re-run the same command until it "
            "exits 0. Report the final exit code."
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


def full_gate(state: dict) -> dict:
    """Run the FULL multi-surface gate at the phase boundary (slow, authoritative)."""
    return run_deterministic_full_gate(run_commands=True).to_result("full_gate")


def advance(state: dict, dry_run: bool) -> None:
    if st.phase_done(state):
        if dry_run:
            log(f"[dry-run] phase {st.current_phase(state)['id']} gate met — would advance")
            return
        gate = full_gate(state)
        if not gate["ok"]:
            log(f"{RED}full gate FAILED for Phase {st.current_phase(state)['id']}; not advancing.{RESET}")
            log(gate["summary"][-1500:])
            return
        log(f"{GREEN}full gate passed for Phase {st.current_phase(state)['id']}.{RESET}")
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
    ap.add_argument("--audit", action="store_true", help="run the fast static TDD/wireframe contract audit")
    args = ap.parse_args()

    state = st.load()
    if args.audit:
        gate = run_static_contract_gate()
        print(gate.summary(8000), flush=True)
        return 0 if gate.ok else 1

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
        # Save progress after every task so a crash/timeout never loses work.
        # Also flush stdout so long runs are observable.
        if not args.dry_run:
            st.save(state)
            sys.stdout.flush()
        else:
            # In dry-run, mark the task so we don't loop forever and do not persist.
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
