"""Persistent phase/task state for the autonomous pipeline."""

from __future__ import annotations

import json
import os
import time

STATE_PATH = os.path.join(os.path.dirname(os.path.abspath(__file__)), "phase_status.json")


def load() -> dict:
    with open(STATE_PATH, encoding="utf-8") as f:
        return json.load(f)


def save(state: dict) -> None:
    with open(STATE_PATH, "w", encoding="utf-8") as f:
        json.dump(state, f, indent=2, ensure_ascii=False)


def current_phase(state: dict) -> dict:
    for p in state["phases"]:
        if p["id"] == state["current_phase"]:
            return p
    raise KeyError("current_phase not found")


def pending_tasks(state: dict) -> list[dict]:
    phase = current_phase(state)
    return [t for t in phase["tasks"] if t["status"] == "PENDING"]


def next_task(state: dict) -> dict | None:
    pending = pending_tasks(state)
    return pending[0] if pending else None


def mark_task(state: dict, task_id: str, status: str, note: str = "") -> dict:
    phase = current_phase(state)
    for t in phase["tasks"]:
        if t["id"] == task_id:
            t["status"] = status
            t["updated_at"] = time.time()
            if note:
                t["note"] = note
            return t
    raise KeyError(f"task {task_id} not in phase {phase['id']}")


def phase_done(state: dict) -> bool:
    return all(t["status"] in ("DONE", "SKIPPED") for t in current_phase(state)["tasks"])


def advance_phase(state: dict) -> dict:
    phase = current_phase(state)
    phase["status"] = "DONE"
    nxt = state["current_phase"] + 1
    if nxt < len(state["phases"]):
        state["current_phase"] = nxt
        state["phases"][nxt]["status"] = "IN_PROGRESS"
    return state
