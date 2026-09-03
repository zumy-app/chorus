"""Chorus CrewAI pipeline package.

Modern CrewAI v1.15.18 architecture:
- crew/agents.json          # JSON-first agent definitions
- crew/tasks.json           # Task templates
- crew/crew.json            # Crew orchestration config
- crew/models.py            # Pydantic structured output models
- crew/flow.py              # CrewAI Flow-based orchestrator (@start, @listen)
- crew/roles.py             # Legacy role definitions (backward compat)
- crew/state.py             # Phase/task state persistence
- crew/phase_status.json    # Current build phase state
"""

from crew.models import GapAnalysis, TaskResult, WireframeParityReport, ReleaseGateCheck, RequirementTrace
from crew.state import load, save, current_phase, pending_tasks, next_task, mark_task, phase_done, advance_phase
from crew.roles import agent_summaries

__all__ = [
    "GapAnalysis",
    "TaskResult",
    "WireframeParityReport",
    "ReleaseGateCheck",
    "RequirementTrace",
    "load",
    "save",
    "current_phase",
    "pending_tasks",
    "next_task",
    "mark_task",
    "phase_done",
    "advance_phase",
    "agent_summaries",
]
