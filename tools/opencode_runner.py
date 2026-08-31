"""Bridge tool: lets CrewAI delegate real implementation work to opencode.

opencode is the "hands": it reads files, edits them, runs commands, writes git.
CrewAI is the "brain": it decides, plans, reviews, and signs off.

This wrapper shells out to `opencode run` non-interactively, attaches the
task as a session file, runs it in the repo cwd, and captures the structured
JSON result so the QA/reviewer agents can act on it.
"""

from __future__ import annotations

import json
import os
import subprocess
import sys
import tempfile
import time
from typing import Any

from crewai.tools import BaseTool
from pydantic import BaseModel, Field

REPO_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
JOBS_DIR = os.path.join(REPO_ROOT, "agent_jobs")

# Agent name (suffix) -> opencode subagent to route to. Falls back to "general".
AGENT_ROUTE = {
    "backend": "backend-engineer",
    "frontend": "frontend-engineer",
    "mobile": "mobile-engineer",
    "infra": "sre-engineer",
    "design": "general",
    "admin": "general",
    "review": "reviewer",
    "teacher": "teacher",
}

# Must match the model the interactive session runs on so headless subagents
# use an already-configured provider. Override via env OPENCODE_MODEL.
DEFAULT_MODEL = os.environ.get("OPENCODE_MODEL", "opencode-go/deepseek-v4-flash-vision-exp")


class OpencodeRunSchema(BaseModel):
    task_prompt: str = Field(
        ...,
        description="Human-readable, self-contained implementation prompt for opencode. "
        "Include explicit file paths and acceptance criteria.",
    )
    agent_suffix: str = Field(
        default="general",
        description="Domain suffix: backend|frontend|mobile|infra|design|admin|review|teacher.",
    )
    job_id: str = Field(
        default="",
        description="Optional job id used for the task/result filenames. Auto-generated if empty.",
    )


class OpencodeRunnerTool(BaseTool):
    name: str = "Opencode Runner"
    description: str = (
        "Delegates an implementation/verification prompt to the opencode agent, "
        "which edits files, runs builds/tests, and produces a structured result "
        "JSON with stdout, stderr, and exit code. Use this to perform ALL real "
        "code changes and command execution. Return the structured result."
    )
    args_schema: type[BaseModel] = OpencodeRunSchema

    def _run(self, task_prompt: str, agent_suffix: str = "general", job_id: str = "") -> str:
        job_id = job_id or f"job_{int(time.time())}_{agent_suffix}"
        task_file = os.path.join(JOBS_DIR, f"{job_id}_task.md")
        result_file = os.path.join(JOBS_DIR, f"{job_id}_result.json")
        os.makedirs(JOBS_DIR, exist_ok=True)

        with open(task_file, "w", encoding="utf-8") as f:
            f.write(_wrap_task(task_prompt))

        subagent = AGENT_ROUTE.get(agent_suffix, "general")
        route_args: list[str] = []
        # Only pass --agent if it maps to an existing agent; default primary is fine.
        if subagent != "general":
            route_args = ["--agent", subagent]

        cmd = [
            _opencode_binary(),
            "run",
            *route_args,
            "--auto",
            "--format",
            "json",
            "--model",
            DEFAULT_MODEL,
            "--dir",
            REPO_ROOT,
            # Pass the task as the positional message so opencode has a prompt.
            # On Windows the argv limit is ~32k; large prompts are written to a
            # file and referenced by the agent via the file attachment instead.
            _truncate(task_prompt, 20000),
        ]

        started = time.time()
        try:
            proc = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                encoding="utf-8",
                errors="replace",
                timeout=1200,
                cwd=REPO_ROOT,
                shell=False,
            )
        except subprocess.TimeoutExpired:
            result = {
                "job_id": job_id,
                "ok": False,
                "error": "opencode timed out after 1200s",
                "stdout": "",
                "stderr": "",
                "exit_code": -1,
                "elapsed": time.time() - started,
                "summary": "",
                "files_changed": [],
            }
            _write_result(result_file, result)
            return _pretty(result)

        elapsed = time.time() - started
        summary, files_changed = _parse_json_events(proc.stdout or "")

        result = {
            "job_id": job_id,
            "ok": proc.returncode == 0,
            "error": None,
            "stdout": _truncate(proc.stdout or "", 8000),
            "stderr": _truncate(proc.stderr or "", 8000),
            "exit_code": proc.returncode,
            "elapsed": round(elapsed, 2),
            "summary": summary,
            "files_changed": files_changed,
        }
        _write_result(result_file, result)
        return _pretty(result)


def _wrap_task(task_prompt: str) -> str:
    return (
        "# Autonomous Build Task\n\n"
        "You are working inside the Chorus monorepo at the repository root.\n"
        "Your job is to make the requested code changes and verify them.\n"
        "Follow the working-set guardrails in WORKING_SET.md when one exists.\n"
        "Run the project's own test/build commands to confirm your changes.\n"
        "Report precisely: what you changed, which commands you ran, and the "
        "exact pass/fail + exit code for each.\n\n"
        "## Task\n\n"
        f"{task_prompt}\n"
    )


def _opencode_binary() -> str:
    """Return the callable opencode entrypoint for this OS."""
    # Resolve the same way the shell does; prefer a real file (not .ps1 dir).
    if os.name == "nt":
        # npm ships opencode.cmd on Windows
        for candidate in ("opencode.cmd", "opencode.bat", "opencode"):
            path = _which(candidate)
            if path:
                return path
        return "opencode.cmd"
    return "opencode"


def _which(name: str) -> str | None:
    import shutil

    resolved = shutil.which(name)
    if resolved and os.path.splitext(resolved)[1] != ".ps1":
        return resolved
    return None


def _parse_json_events(stdout: str) -> tuple[str, list[str]]:
    """Extract a text summary + changed file list from opencode --format json.

    opencode emits assistant text at ``part.text`` (not a top-level ``message``),
    so read the content from there and defensively across the shapes it can emit.
    Without this, the QA "PASS" verdict is silently dropped and the orchestrator
    treats every green build as a failed verification.
    """
    summary_parts: list[str] = []
    files_changed: list[str] = []
    for line in stdout.splitlines():
        line = line.strip()
        if not line.startswith("{"):
            continue
        try:
            evt = json.loads(line)
        except json.JSONDecodeError:
            continue
        file = evt.get("file") or evt.get("path")
        if file and file not in files_changed:
            files_changed.append(file)
        if evt.get("type") == "text":
            msg = _extract_message(evt)
            if msg:
                summary_parts.append(msg)
    summary = "\n".join(summary_parts)
    return _truncate(summary, 2500), files_changed


def _extract_message(evt: dict) -> str:
    """Pull the assistant message out of an opencode JSONL event.

    Handles the canonical ``{"type":"text","part":{"text":...}}`` shape plus
    legacy/flat variants (``message``, ``text``, or a nested part object).
    """
    candidates = (
        evt.get("message"),
        evt.get("text"),
        (evt.get("part") or {}).get("text") if isinstance(evt.get("part"), dict) else None,
        (evt.get("part") or {}).get("message") if isinstance(evt.get("part"), dict) else None,
    )
    for val in candidates:
        if isinstance(val, str):
            return val
        if isinstance(val, dict):
            for key in ("text", "content", "message"):
                inner = val.get(key)
                if isinstance(inner, str):
                    return inner
    return ""


def _write_result(path: str, data: dict[str, Any]) -> None:
    try:
        with open(path, "w", encoding="utf-8") as f:
            json.dump(data, f, indent=2, ensure_ascii=False)
    except OSError:
        pass


def _truncate(text: str, limit: int) -> str:
    if len(text) <= limit:
        return text
    return text[:limit] + f"\n...[truncated {len(text) - limit} chars]"


def _pretty(result: dict[str, Any]) -> str:
    return json.dumps(result, indent=2, ensure_ascii=False)


if __name__ == "__main__":
    # Manual smoke test: `python tools/opencode_runner.py "<prompt>"`
    prompt = sys.argv[1] if len(sys.argv) > 1 else "echo hello and report the current git branch."
    print(OpencodeRunnerTool()._run(prompt, job_id="smoke"))
