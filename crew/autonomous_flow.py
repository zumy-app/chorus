"""Autonomous CrewAI Flow for implementing all missing Chorus features (Phases 5-10).

This Flow uses the OpencodeRunnerTool to delegate real implementation work to
opencode subagents. CrewAI is the brain (plans, reviews, gates), opencode is
the hands (edits files, runs builds, executes commands).

Role responsibilities:
- analyst: requirements traceability, gap analysis
- product_manager: prioritization, scope
- backend_engineer: Go/Gin backend implementation
- frontend_engineer: React/Vite web implementation
- mobile_engineer: Expo RN mobile implementation
- qa_engineer: designs + executes test suites for every feature
- test_engineer: writes + maintains automated tests
- teacher: reviews learning content quality (captions, translations)
- sre: performance, scalability, NFRs, load testing, infrastructure
- reviewer: code review, release gate verification

Usage:
  python -m crew.autonomous_flow                  # run all phases 5-10
  python -m crew.autonomous_flow --phase 5        # run specific phase
  python -m crew.autonomous_flow --dry-run        # print plan only
"""

from __future__ import annotations

import argparse
import json
import os
import sys
import time
from datetime import datetime
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from crewai import Agent, Crew, Task, Process
from crewai.flow.flow import Flow, listen, start
from dotenv import load_dotenv

from crew.models import GapAnalysis, TaskResult, WireframeParityReport, ReleaseGateCheck
from tools.opencode_runner import OpencodeRunnerTool

load_dotenv()

REPO_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
CREW_DIR = os.path.join(REPO_ROOT, "crew")
PLAN_PATH = os.path.join(CREW_DIR, "autonomous_plan.md")
STATE_PATH = os.path.join(CREW_DIR, "phase5plus_status.json")


# Role -> opencode agent suffix mapping
ROLE_SUFFIX = {
    "analyst": "analyst",
    "product_manager": "admin",
    "backend_engineer": "backend",
    "frontend_engineer": "frontend",
    "mobile_engineer": "mobile",
    "qa_engineer": "qa",
    "test_engineer": "test",
    "teacher": "teacher",
    "sre": "infra",
    "reviewer": "review",
}


def log(msg: str) -> None:
    now = datetime.now()
    print(f"[{now.hour:02d}:{now.minute:02d}:{now.second:02d}] {msg}", flush=True)


def load_state() -> dict:
    with open(STATE_PATH, encoding="utf-8") as f:
        return json.load(f)


def save_state(state: dict) -> None:
    with open(STATE_PATH, "w", encoding="utf-8") as f:
        json.dump(state, f, indent=2, ensure_ascii=False)


def get_phase_tasks(phase_id: int) -> list[dict]:
    state = load_state()
    for p in state["phases"]:
        if p["id"] == phase_id:
            return [t for t in p["tasks"] if t["status"] == "PENDING"]
    return []


def mark_task_done(task_id: str, note: str = "") -> None:
    state = load_state()
    for p in state["phases"]:
        for t in p["tasks"]:
            if t["id"] == task_id:
                t["status"] = "DONE"
                t["updated_at"] = time.time()
                if note:
                    t["note"] = note[:2000]
                save_state(state)
                return
    log(f"WARNING: task {task_id} not found in state")


def mark_task_failed(task_id: str, note: str = "") -> None:
    state = load_state()
    for p in state["phases"]:
        for t in p["tasks"]:
            if t["id"] == task_id:
                t["status"] = "FAILED"
                t["updated_at"] = time.time()
                if note:
                    t["note"] = note[:2000]
                save_state(state)
                return


def phase_complete(phase_id: int) -> bool:
    state = load_state()
    for p in state["phases"]:
        if p["id"] == phase_id:
            return all(t["status"] in ("DONE", "SKIPPED") for t in p["tasks"])
    return False


def advance_to_phase(phase_id: int) -> None:
    state = load_state()
    for p in state["phases"]:
        if p["id"] == phase_id:
            p["status"] = "DONE"
        elif p["id"] == phase_id + 1:
            p["status"] = "IN_PROGRESS"
    state["current_phase"] = phase_id + 1
    save_state(state)


def build_task_prompt(task: dict, phase_id: int) -> str:
    """Build a detailed prompt for the opencode runner."""
    plan_text = Path(PLAN_PATH).read_text(encoding="utf-8")
    return f"""# PHASE {phase_id}: TASK {task["id"]} — {task["name"]}

## Context
You are working on the Chorus multilingual real-time messenger.
Stack: Go+Gin backend, React+TypeScript+Vite frontend, Expo React Native mobile.
PostgreSQL is the durable source of truth. Redis handles cache + pub/sub + WebSocket registry.

## Your Assignment
Implement: {task["name"]}

## Detailed Plan
Read crew/autonomous_plan.md for the full phase plan. Your task is section {task["id"]}.

## Requirements
1. Read the relevant wireframes in wireframes/ directory.
2. Read REQUIREMENTS_MASTER.md for the requirement context.
3. Inspect existing code before making changes.
4. Implement the feature across backend (Go), frontend (React/TS), and mobile (Expo RN) as needed.
5. Write tests for the feature.
6. Run the affected layer's build and tests:
   - Backend: cd backend && go build ./... && go test ./...
   - Frontend: cd frontend && npm test && npm run build
   - Mobile: cd mobile && npm test
7. Report: files changed, commands run, exact exit codes, and a summary.

## Constraints
- Never write secrets or touch .env* files.
- Never touch agent_jobs/, crew/, tools/, or data/ directories.
- Mobile-first, web parity (NFR-22).
- No stubs/placeholders in shipped UX.
- Follow existing code conventions.
"""


def delegate_to_agent(task: dict, phase_id: int, dry_run: bool = False) -> dict:
    """Delegate a task to the appropriate opencode subagent."""
    role = task.get("role", "backend_engineer")
    suffix = ROLE_SUFFIX.get(role, "general")
    prompt = build_task_prompt(task, phase_id)

    if dry_run:
        log(f"[DRY-RUN] Would delegate task {task["id"]} to {role} (suffix={suffix})")
        log(f"  Prompt: {prompt[:200]}...")
        return {"ok": True, "summary": "dry-run", "exit_code": 0}

    log(f"Delegating task {task["id"]} to {role} (opencode suffix={suffix})...")
    tool = OpencodeRunnerTool()
    raw = tool._run(prompt, agent_suffix=suffix, job_id=f"phase{phase_id}_{task["id"]}")

    try:
        result = json.loads(raw)
    except (json.JSONDecodeError, TypeError):
        result = {"ok": False, "summary": str(raw)[:2000], "exit_code": 1}

    return result


# =====================================================================
# CrewAI Flow — Phase by Phase Autonomous Execution
# =====================================================================

class AutonomousBuildFlow(Flow):
    """CrewAI Flow that autonomously implements all missing Chorus features."""

    # stream = True  # disabled for blocking execution

    def _run_phase(self, phase_id: int, dry_run: bool = False) -> dict:
        """Execute all tasks in a phase."""
        state = load_state()
        phase = None
        for p in state["phases"]:
            if p["id"] == phase_id:
                phase = p
                break
        if not phase:
            log(f"ERROR: Phase {phase_id} not found")
            return {"phase": phase_id, "status": "ERROR"}

        log(f"\n{'='*60}")
        log(f"PHASE {phase_id}: {phase["name"]}")
        log(f"Gate: {phase.get("gate", "")}")
        log(f"{'='*60}")

        tasks = get_phase_tasks(phase_id)
        if not tasks:
            log(f"Phase {phase_id} has no pending tasks — already complete")
            return {"phase": phase_id, "status": "ALREADY_DONE"}

        results = []
        for task in tasks:
            log(f"\n--- Task {task["id"]}: {task["name"]} ---")
            result = delegate_to_agent(task, phase_id, dry_run=dry_run)
            results.append({"task_id": task["id"], "result": result})

            if result.get("ok", False):
                mark_task_done(task["id"], result.get("summary", ""))
                log(f"Task {task["id"]} DONE")
            else:
                mark_task_failed(task["id"], result.get("stderr", result.get("summary", "")))
                log(f"Task {task["id"]} FAILED: {result.get("summary", "")[:200]}")

            # Save state after every task
            save_state(load_state())

        # Check if phase is complete
        if phase_complete(phase_id):
            log(f"\nPhase {phase_id} COMPLETE — all tasks done")
            if phase_id < 10:
                advance_to_phase(phase_id)
                log(f"Advanced to Phase {phase_id + 1}")
        else:
            failed = [t for t in get_phase_tasks(phase_id)]
            log(f"Phase {phase_id} still has {len(failed)} pending/failed tasks")

        return {"phase": phase_id, "results": results}

    # --- Phase 5: Messaging Parity ---
    @start()
    def phase_5_messaging(self):
        return self._run_phase(5)

    # --- Phase 6: Privacy & Security ---
    @listen(phase_5_messaging)
    def phase_6_privacy(self, prev):
        return self._run_phase(6)

    # --- Phase 7: Audio Calling ---
    @listen(phase_6_privacy)
    def phase_7_audio_call(self, prev):
        return self._run_phase(7)

    # --- Phase 8: Video Calling ---
    @listen(phase_7_audio_call)
    def phase_8_video_call(self, prev):
        return self._run_phase(8)

    # --- Phase 9: SRE Performance ---
    @listen(phase_8_video_call)
    def phase_9_sre(self, prev):
        return self._run_phase(9)

    # --- Phase 10: QA & Release ---
    @listen(phase_9_sre)
    def phase_10_release(self, prev):
        return self._run_phase(10)


# =====================================================================
# Entry Point
# =====================================================================

def run_autonomous(dry_run: bool = False, start_phase: int = 5) -> dict:
    """Run the autonomous build flow."""
    log("=" * 60)
    log("Chorus Autonomous Build Flow — CrewAI v1.15.18")
    log(f"Phases 5-10: Implementing all missing features from wireframes")
    log(f"Dry run: {dry_run}")
    log(f"Start phase: {start_phase}")
    log("=" * 60)

    if not os.path.exists(STATE_PATH):
        log(f"ERROR: {STATE_PATH} not found")
        return {"status": "ERROR"}

    if dry_run:
        # Dry run: just print what would happen
        for phase_id in range(start_phase, 11):
            tasks = get_phase_tasks(phase_id)
            state = load_state()
            phase = next(p for p in state["phases"] if p["id"] == phase_id)
            log(f"\nPhase {phase_id}: {phase["name"]} ({len(tasks)} tasks)")
            for t in tasks:
                log(f"  [{t["id"]}] {t["name"]} -> {t.get("role", "backend_engineer")}")
        return {"status": "DRY_RUN_COMPLETE"}

    # For specific phase: run just that phase
    if start_phase >= 5:
        log(f"Running single phase {start_phase}")
        flow = AutonomousBuildFlow()
        # Run just the specified phase
        result = flow._run_phase(start_phase)
        return {"status": "DONE", "result": str(result)}

    # Full flow: run all phases 5-10
    flow = AutonomousBuildFlow()
    result = flow.kickoff()
    log(f"\nFlow complete: {result}")
    return {"status": "DONE", "result": str(result)}


if __name__ == "__main__":
    ap = argparse.ArgumentParser(description="Autonomous Chorus build (Phases 5-10)")
    ap.add_argument("--dry-run", action="store_true", help="Print plan without executing")
    ap.add_argument("--phase", type=int, default=5, help="Start phase (5-10)")
    ap.add_argument("--only", default=None, help="Run a single task id (e.g. 5.1)")
    args = ap.parse_args()

    if args.only:
        # Run a single task
        state = load_state()
        for p in state['phases']:
            for t in p['tasks']:
                if t['id'] == args.only:
                    log(f'Running single task {args.only}')
                    result = delegate_to_agent(t, p['id'])
                    if result.get('ok', False):
                        mark_task_done(t['id'], result.get('summary', ''))
                        log(f'Task {args.only} DONE')
                    else:
                        mark_task_failed(t['id'], result.get('stderr', result.get('summary', '')))
                        log(f'Task {args.only} FAILED')
                    save_state(load_state())
                    sys.exit(0 if result.get('ok') else 1)
        log(f'Task {args.only} not found')
        sys.exit(1)
    result = run_autonomous(dry_run=args.dry_run, start_phase=args.phase)
    sys.exit(0 if result.get("status") in ("DONE", "DRY_RUN_COMPLETE") else 1)
