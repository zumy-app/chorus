"""Pydantic models for structured outputs in the Chorus CrewAI pipeline."""

from pydantic import BaseModel, Field


class RequirementTrace(BaseModel):
    """Trace from a wireframe requirement to its code implementation."""

    requirement_id: str = Field(description="Requirement ID")
    wireframe_path: str = Field(description="Path to the wireframe file")
    description: str = Field(description="What the requirement specifies")
    frontend_screen: str | None = Field(default=None, description="Frontend screen/route if implemented")
    mobile_screen: str | None = Field(default=None, description="Mobile screen/route if implemented")
    backend_handler: str | None = Field(default=None, description="Backend handler/service if implemented")
    status: str = Field(default="MISSING", description="IMPLEMENTED, MISSING, or PARTIAL")
    notes: str = Field(default="", description="Implementation notes")


class GapAnalysis(BaseModel):
    """Output of an analyst review showing gaps between wireframes and code."""

    total_requirements: int = Field(description="Total requirements found")
    implemented: int = Field(description="Requirements with full code coverage")
    missing: int = Field(description="Requirements with no code implementation")
    partial: int = Field(description="Requirements with partial implementation")
    gaps: list[RequirementTrace] = Field(description="Requirement traces sorted by status")
    summary: str = Field(description="Executive summary")


class TaskResult(BaseModel):
    """Structured result of a completed build task."""

    task_id: str = Field(description="Task identifier")
    phase: int = Field(description="Phase number")
    status: str = Field(description="DONE, FAILED, or SKIPPED")
    files_changed: list[str] = Field(default_factory=list, description="Files modified")
    commands_run: list[str] = Field(default_factory=list, description="Commands executed")
    exit_code: int = Field(default=0, description="Exit code of the last command")
    summary: str = Field(default="", description="Human-readable summary")
    test_results: str | None = Field(default=None, description="Test output")


class WireframeParityReport(BaseModel):
    """QA parity check: every wireframe must have a reachable screen."""

    wireframes_checked: int = Field(description="Number of wireframe files reviewed")
    reachable_screens: int = Field(description="Screens with proper navigation routes")
    missing_screens: list[str] = Field(default_factory=list, description="Wireframes without screens")
    broken_navigation: list[str] = Field(default_factory=list, description="Routes that fail to render")
    verification_passed: bool = Field(description="True if all wireframes have reachable screens")
    summary: str = Field(description="Detailed parity report")


class ReleaseGateCheck(BaseModel):
    """Release gate verification result."""

    gate_name: str = Field(description="Name of the gate")
    checks: dict[str, bool] = Field(description="Map of check name to pass/fail")
    all_passed: bool = Field(description="True only if every check passed")
    failed_checks: list[str] = Field(default_factory=list, description="Names of failed checks")
    summary: str = Field(description="Gate assessment summary")
