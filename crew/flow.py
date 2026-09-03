"""CrewAI v1.15.18 Flow orchestrator for the Chorus autonomous build pipeline."""

from __future__ import annotations
import json
import os
import sys
from datetime import datetime
from typing import Any

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from crewai import Agent, Crew, Task, Process
from crewai.flow.flow import Flow, listen, start
from dotenv import load_dotenv

from crew.models import GapAnalysis, TaskResult, WireframeParityReport, ReleaseGateCheck
from crew.state import load as load_state
from crew.state import save as save_state
from crew.state import next_task as get_next_task
from crew.state import phase_done
from crew.state import advance_phase

load_dotenv()

REPO_ROOT = os.path.dirname(os.path.abspath(__file__))
CREW_DIR = REPO_ROOT
STATE_PATH = os.path.join(REPO_ROOT, "crew", "phase_status.json")


def log(msg: str) -> None:
    now = datetime.now()
    print(f"[{now.hour:02d}:{now.minute:02d}:{now.second:02d}] {msg}", flush=True)


def load_config(path: str) -> dict:
    with open(path, encoding="utf-8") as f:
        return json.load(f)


def make_agent(name: str, cfg: dict) -> Agent:
    return Agent(
        role=cfg["role"],
        goal=cfg["goal"],
        backstory=cfg.get("backstory", ""),
        verbose=cfg.get("verbose", True),
        allow_delegation=cfg.get("allow_delegation", False),
        memory=cfg.get("memory", True),
    )


# Phase task definitions (from phase_status.json)
PHASE_TASKS: dict[int, list[dict]] = {
    0: [
        {"id": "P0-1", "name": "Backend build + go test green", "role": "backend_engineer"},
        {"id": "P0-2", "name": "Frontend build + npm test green", "role": "frontend_engineer"},
        {"id": "P0-3", "name": "Mobile jest green", "role": "mobile_engineer"},
        {"id": "P0-4", "name": "Docker compose dev valid", "role": "sre"},
        {"id": "P0-5", "name": "Docs + phase tracking", "role": "analyst"},
        {"id": "P0-6", "name": "Remove dead artifacts", "role": "reviewer"},
        {"id": "P0-7", "name": "Canonical run commands", "role": "reviewer"},
    ],
    1: [
        {"id": "1.1", "name": "Home button link dashboard", "role": "frontend_engineer"},
        {"id": "1.2", "name": "Learn dashboard wiring", "role": "frontend_engineer"},
        {"id": "1.3", "name": "Translation accuracy", "role": "backend_engineer"},
    ],
    2: [
        {"id": "2.1", "name": "SRS vocabulary engine", "role": "backend_engineer"},
        {"id": "2.2", "name": "Lesson & CEFR content", "role": "teacher"},
    ],
    3: [
        {"id": "3.1", "name": "L4 load balancer", "role": "sre"},
        {"id": "3.2", "name": "Redis scaling & observability", "role": "sre"},
    ],
    4: [
        {"id": "MKT-1", "name": "Marketplace nav wiring", "role": "mobile_engineer"},
        {"id": "MKT-2", "name": "Marketplace screens", "role": "frontend_engineer"},
        {"id": "MKT-3", "name": "Confirm booking + polish", "role": "frontend_engineer"},
        {"id": "MKT-QA", "name": "QA parity gate", "role": "qa_engineer"},
        {"id": "QA-FUNC", "name": "QA functional suite", "role": "test_engineer"},
    ],
}
# ---------------------------------------------------------------------------
# ChorusBuildFlow - CrewAI Flow orchestrating all phases
# ---------------------------------------------------------------------------

class ChorusBuildFlow(Flow):
    """CrewAI Flow orchestrating the Chorus autonomous multi-phase build."""

    stream = True

    def _run_crew_for_task(self, task_def: dict, agents_config: dict) -> dict:
        """Run a single task via a Crew with the appropriate agent."""
        role = task_def.get("role", "backend_engineer")
        agent_cfg = agents_config.get(role, agents_config["backend_engineer"])
        agent = make_agent(role, agent_cfg)

        task = Task(
            description=(
                f"Implement task {task_def["id"]}: {task_def["name"]}.\n"
                f"Refer to wireframes/ for the spec and REQUIREMENTS_MASTER.md for requirements.\n"
                f"Inspect existing code first. Write production code. Run tests. Report exit codes."
            ),
            expected_output=f"Task {task_def["id"]} complete. Files changed, tests pass, summary provided.",
            agent=agent,
            name=task_def["id"],
        )

        crew = Crew(
            agents=[agent],
            tasks=[task],
            process=Process.sequential,
            verbose=True,
            memory=True,
            planning=True,
            cache=True,
        )

        result = crew.kickoff()
        raw = result.raw if hasattr(result, "raw") else str(result)
        return {"id": task_def["id"], "result": raw, "exit_code": 0 if raw else 1}

    @start()
    def phase_0_foundation(self):
        """Phase 0: Foundation & Green Baseline."""
        log("=== PHASE 0: Foundation & Green Baseline ===")
        agents_config = load_config(os.path.join(CREW_DIR, "crew", "agents.json"))
        state = load_state()
        results = []
        for t in PHASE_TASKS[0]:
            r = self._run_crew_for_task(t, agents_config)
            results.append(r)
            try:
                mark_state = load_state()
                for pt in mark_state["phases"][0]["tasks"]:
                    if pt["id"] == t["id"]:
                        pt["status"] = "DONE" if r["exit_code"] == 0 else "FAILED"
                        pt["updated_at"] = datetime.now().timestamp()
                save_state(mark_state)
            except Exception as e:
                log(f"State update warning: {e}")
        return {"phase": 0, "results": results}
    @listen(phase_0_foundation)
    def phase_1_core(self, prev_result):
        """Phase 1: Launch-Blocking Core."""
        log("=== PHASE 1: Launch-Blocking Core (P0) ===")
        agents_config = load_config(os.path.join(CREW_DIR, "crew", "agents.json"))
        state = load_state()
        results = []
        for t in PHASE_TASKS[1]:
            r = self._run_crew_for_task(t, agents_config)
            results.append(r)
        return {"phase": 1, "results": results, "prev": prev_result}

    @listen(phase_1_core)
    def phase_2_learning(self, prev_result):
        """Phase 2: Learning & Vocabulary."""
        log("=== PHASE 2: Learning & Vocabulary ===")
        agents_config = load_config(os.path.join(CREW_DIR, "crew", "agents.json"))
        results = []
        for t in PHASE_TASKS[2]:
            r = self._run_crew_for_task(t, agents_config)
            results.append(r)
        return {"phase": 2, "results": results, "prev": prev_result}

    @listen(phase_2_learning)
    def phase_3_scaling(self, prev_result):
        """Phase 3: Scaling & Infrastructure."""
        log("=== PHASE 3: Scaling & Infrastructure ===")
        agents_config = load_config(os.path.join(CREW_DIR, "crew", "agents.json"))
        results = []
        for t in PHASE_TASKS[3]:
            r = self._run_crew_for_task(t, agents_config)
            results.append(r)
        return {"phase": 3, "results": results, "prev": prev_result}

    @listen(phase_3_scaling)
    def phase_4_marketplace(self, prev_result):
        """Phase 4: Marketplace & Production Readiness."""
        log("=== PHASE 4: Marketplace & Production Readiness ===")
        agents_config = load_config(os.path.join(CREW_DIR, "crew", "agents.json"))
        results = []
        for t in PHASE_TASKS[4]:
            r = self._run_crew_for_task(t, agents_config)
            results.append(r)
        return {"phase": 4, "results": results, "prev": prev_result}

    @listen(phase_4_marketplace)
    def production_release(self, prev_result):
        """Final production release gate."""
        log("=== PRODUCTION RELEASE GATE ===")
        agents_config = load_config(os.path.join(CREW_DIR, "crew", "agents.json"))
        tasks_config = load_config(os.path.join(CREW_DIR, "crew", "tasks.json"))

        reviewer = make_agent("reviewer", agents_config["reviewer"])
        gate_task = Task(
            description=(
                "Run the production release gate:\n"
                "1. All phase tasks DONE in phase_status.json\n"
                "2. Backend builds green (go build ./...)\n"
                "3. Backend tests pass (go test ./...)\n"
                "4. Frontend builds (npm run build)\n"
                "5. Frontend tests pass (npm test)\n"
                "6. Mobile tests pass (npm test)\n"
                "7. Security scan passes\n"
                "8. Go/No-Go checklist complete\n"
                "Output structured ReleaseGateCheck."
            ),
            expected_output="ReleaseGateCheck with pass/fail for each check and overall verdict.",
            agent=reviewer,
            name="release_gate",
        )

        gate_crew = Crew(
            agents=[reviewer],
            tasks=[gate_task],
            process=Process.sequential,
            verbose=True,
            memory=True,
        )
        result = gate_crew.kickoff()
        raw = result.raw if hasattr(result, "raw") else str(result)
        log(f"Release gate: {raw[:500]}")
        return {"gate": "release", "result": raw, "prev": prev_result}


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------

def run_flow() -> dict:
    """Run the ChorusBuildFlow."""
    log("Chorus Autonomous Build Flow starting (CrewAI v1.15.18)")
    if not os.path.exists(STATE_PATH):
        log(f"ERROR: {STATE_PATH} not found")
        return {"status": "ERROR", "message": f"Missing {STATE_PATH}"}
    flow = ChorusBuildFlow()
    result = flow.kickoff()
    log(f"Flow complete: {result}")
    return {"status": "DONE", "result": str(result)}


if __name__ == "__main__":
    run_flow()
