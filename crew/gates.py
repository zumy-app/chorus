"""Deterministic quality gates for the autonomous Chorus crew.

The agents can write convincing summaries. These gates decide whether the
workspace actually meets the contract before state is allowed to move to DONE.
"""

from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
import os
import re
import shutil
import subprocess


REPO_ROOT = Path(__file__).resolve().parents[1]


@dataclass
class CheckResult:
    name: str
    ok: bool
    details: str = ""
    command: str | None = None
    exit_code: int | None = None


@dataclass
class GateResult:
    name: str
    checks: list[CheckResult]

    @property
    def ok(self) -> bool:
        return all(c.ok for c in self.checks)

    def summary(self, limit: int = 6000) -> str:
        lines = [f"{self.name}: {'PASS' if self.ok else 'FAIL'}"]
        for check in self.checks:
            status = "PASS" if check.ok else "FAIL"
            suffix = ""
            if check.exit_code is not None:
                suffix += f" exit={check.exit_code}"
            if check.command:
                suffix += f" [{check.command}]"
            lines.append(f"- {status}: {check.name}{suffix}")
            if check.details:
                lines.append(f"  {check.details.strip()[:1200]}")
        text = "\n".join(lines)
        return text if len(text) <= limit else text[:limit] + f"\n...[truncated {len(text) - limit} chars]"

    def to_result(self, job_id: str) -> dict:
        return {
            "job_id": job_id,
            "ok": self.ok,
            "error": None if self.ok else "gate failed",
            "stdout": self.summary(8000),
            "stderr": "" if self.ok else self.summary(8000),
            "exit_code": 0 if self.ok else 1,
            "elapsed": 0.0,
            "summary": self.summary(2500),
            "files_changed": [],
        }


def pass_check(name: str, details: str = "") -> CheckResult:
    return CheckResult(name=name, ok=True, details=details)


def fail_check(name: str, details: str = "") -> CheckResult:
    return CheckResult(name=name, ok=False, details=details)


def read_rel(path: str) -> str:
    return (REPO_ROOT / path).read_text(encoding="utf-8", errors="replace")


def require_file(path: str, patterns: tuple[str, ...] = ()) -> list[CheckResult]:
    target = REPO_ROOT / path
    if not target.exists():
        return [fail_check(f"{path} exists", "required by TDD rescue and release gating")]

    checks = [pass_check(f"{path} exists")]
    text = target.read_text(encoding="utf-8", errors="replace")
    missing = [p for p in patterns if p not in text]
    if missing:
        checks.append(fail_check(f"{path} contains required contract markers", "missing: " + ", ".join(missing)))
    else:
        checks.append(pass_check(f"{path} contains required contract markers"))
    return checks


def run_command(name: str, command: str, timeout: int = 1200) -> CheckResult:
    try:
        proc = subprocess.run(
            command,
            shell=True,
            capture_output=True,
            text=True,
            encoding="utf-8",
            errors="replace",
            cwd=REPO_ROOT,
            timeout=timeout,
        )
    except subprocess.TimeoutExpired:
        return CheckResult(name=name, ok=False, command=command, exit_code=-1, details=f"timed out after {timeout}s")

    out = (proc.stdout or "").strip()
    err = (proc.stderr or "").strip()
    details = (err or out)[-2500:]
    return CheckResult(name=name, ok=proc.returncode == 0, command=command, exit_code=proc.returncode, details=details)


def run_static_contract_gate() -> GateResult:
    checks: list[CheckResult] = []
    checks.extend(run_test_quality_contract().checks)
    checks.extend(run_learning_contract().checks)
    checks.extend(run_wireframe_parity_gate().checks)
    return GateResult("static contract gate", checks)


def run_test_quality_contract() -> GateResult:
    """Reject hollow tests before they can justify DONE state."""

    checks: list[CheckResult] = []
    checks.extend(require_file("docs/TDD_RESCUE_SPEC.md", ("No phase `DONE`", "TC-LEARN-01", "TC-TUTOR-01")))
    checks.extend(require_file("docs/QA_CRITIQUE_AND_IMPROVEMENTS.md", ("C-01", "C-02", "C-05", "AVD")))
    checks.extend(
        require_file(
            "e2e/acceptance/tests/p0-features.ts",
            ("TC-LEARN-04", "/api/v1/learning/sessions/start", "/api/v1/teachers/apply", "/api/v1/teachers/browse"),
        )
    )
    checks.extend(
        require_file(
            "e2e/tests/20-comprehensive-two-user.spec.ts",
            ("@critical", "waitForTranslation", "critical: true", "openAITutor", "Durability check"),
        )
    )
    checks.extend(
        require_file(
            "e2e/tests/21-learning-journey.spec.ts",
            ("C-02", "/learn/placement", "/learn/session", "/learn/real-talk", "learn-monthly"),
        )
    )
    checks.extend(
        require_file(
            "e2e/tests/22-settings-privacy.spec.ts",
            ("C-03", "/blocks", "/reports", "2FA"),
        )
    )
    checks.extend(
        require_file(
            "e2e/tests/23-marketplace-e2e.spec.ts",
            ("C-04", "/tutors", "waitForResponse", "book-trial", "confirm-booking", "payout"),
        )
    )
    checks.extend(
        require_file(
            "e2e/tests/24-teacher-apply.spec.ts",
            ("C-05", "/become-teacher", "/teachers/apply", "Status: pending"),
        )
    )
    checks.extend(require_file("e2e/playwright.config.ts", ("mobile-chrome", "E2E_AVD")))
    checks.extend(require_file("mobile/detox.config.js", ("android.emu.debug", "Pixel_7_API_34")))
    checks.extend(require_file("e2e/wdio.conf.ts", ("UiAutomator2", "emulator-5554")))

    checks.append(_no_legacy_external_accounts())
    checks.extend(_critical_specs_are_hard_fail())
    checks.append(_acceptance_suite_is_real_http())
    checks.append(_acceptance_runner_executes_tests())
    return GateResult("test quality contract", checks)


def _no_legacy_external_accounts() -> CheckResult:
    users_file = REPO_ROOT / "e2e/fixtures/users.ts"
    if not users_file.exists():
        return fail_check("E2E users fixture exists", "missing e2e/fixtures/users.ts")
    text = users_file.read_text(encoding="utf-8", errors="replace")
    bad = sorted(set(re.findall(r"[\w.+-]+@gmail\.com|Demor@cer1|LEGACY|legacy", text, flags=re.IGNORECASE)))
    if bad:
        return fail_check(
            "E2E fixtures use only deterministic dev accounts",
            "remove external/legacy credentials from e2e/fixtures/users.ts: " + ", ".join(bad[:8]),
        )
    return pass_check("E2E fixtures use only deterministic dev accounts")


def _critical_specs_are_hard_fail() -> list[CheckResult]:
    checks: list[CheckResult] = []
    critical_files = [
        "e2e/tests/20-comprehensive-two-user.spec.ts",
        "e2e/tests/23-marketplace-e2e.spec.ts",
        "e2e/tests/24-teacher-apply.spec.ts",
    ]
    soft_re = re.compile(r"\bsoft\b|console\.warn|\.catch\s*\([^)]*console\.warn|soft-pass", re.IGNORECASE | re.DOTALL)
    for path in critical_files:
        target = REPO_ROOT / path
        if not target.exists():
            continue
        text = target.read_text(encoding="utf-8", errors="replace")
        if soft_re.search(text):
            checks.append(
                fail_check(
                    f"{path} has no soft-pass catches",
                    "critical acceptance tests must throw on missing UI/API behavior; remove console.warn/soft-pass fallbacks",
                )
            )
        else:
            checks.append(pass_check(f"{path} has no soft-pass catches"))

        if "readFileSync" in text or "fs.readFile" in text:
            checks.append(
                fail_check(
                    f"{path} does not use file-content assertions as acceptance proof",
                    "file-content checks can remain in smoke tests, but critical journeys must drive the app and APIs",
                )
            )
        else:
            checks.append(pass_check(f"{path} does not use file-content assertions as acceptance proof"))

        if path.endswith("20-comprehensive-two-user.spec.ts"):
            swallowed_critical = re.search(r"try\s*\{[\s\S]*?critical:\s*true[\s\S]*?\}\s*catch", text) or "critical soft" in text.lower()
            if swallowed_critical:
                checks.append(
                    fail_check(
                        f"{path} does not swallow critical translation failures",
                        "waitForTranslation(..., { critical: true }) must be outside try/catch or must rethrow",
                    )
                )
            else:
                checks.append(pass_check(f"{path} does not swallow critical translation failures"))
    return checks


def _acceptance_suite_is_real_http() -> CheckResult:
    path = REPO_ROOT / "e2e/acceptance/tests/p0-features.ts"
    if not path.exists():
        return fail_check("acceptance suite uses real HTTP", "missing e2e/acceptance/tests/p0-features.ts")
    text = path.read_text(encoding="utf-8", errors="replace")
    if "route.fulfill" in text or "page.route" in text or "readFileSync" in text:
        return fail_check("acceptance suite uses real HTTP", "acceptance tests must hit the running stack, not mocks or source files")
    return pass_check("acceptance suite uses real HTTP")


def _acceptance_runner_executes_tests() -> CheckResult:
    path = REPO_ROOT / "e2e/acceptance/run.ts"
    if not path.exists():
        return fail_check("acceptance runner executes tests", "missing e2e/acceptance/run.ts")
    text = path.read_text(encoding="utf-8", errors="replace")
    required = [
        (re.search(r"\bawait\s+buildCtx\s*\(", text), "build the real fixture context"),
        (re.search(r"\bawait\s+runSuite\s*\(", text), "invoke the acceptance test cases"),
        ("process.argv" in text, "read red/green mode from CLI args"),
        (re.search(r"mode\s*===\s*['\"]green|case\s+['\"]green", text), "enforce green-mode semantics"),
        ("process.exit(1)" in text or re.search(r"process\.exitCode\s*=\s*1", text), "fail the command when the contract is red"),
    ]
    missing = [label for ok, label in required if not ok]
    has_entrypoint = re.search(r"\bmain\s*\(\s*\)\s*\.catch|\bawait\s+main\s*\(", text)
    if missing or not has_entrypoint:
        details = []
        if missing:
            details.append("missing executable semantics: " + ", ".join(missing))
        if not has_entrypoint:
            details.append("missing main() entrypoint; a module with only function definitions can exit 0 without running tests")
        return fail_check("acceptance runner executes tests", "; ".join(details))
    return pass_check("acceptance runner executes tests")


def env_flag(name: str) -> bool:
    return os.getenv(name, "").strip().lower() in {"1", "true", "yes", "on"}


def release_acceptance_commands() -> list[tuple[str, str, int]]:
    commands = [
        ("acceptance green", "cd e2e\\acceptance && npm run acceptance:green", 1800),
    ]
    if env_flag("CHORUS_RUN_PLAYWRIGHT"):
        commands.append(
            (
                "critical playwright e2e",
                "cd e2e && npx playwright test 20-comprehensive-two-user.spec.ts 21-learning-journey.spec.ts 23-marketplace-e2e.spec.ts 24-teacher-apply.spec.ts",
                2400,
            )
        )
    return commands


def append_command_checks(checks: list[CheckResult], commands: list[tuple]) -> None:
    for spec in commands:
        name = spec[0]
        command = spec[1]
        timeout = spec[2] if len(spec) > 2 else 1200
        checks.append(run_command(name, command, timeout=timeout))


def run_learning_contract() -> GateResult:
    """Static checks for the Learn dashboard, placement, drills and Sparky flows."""

    checks: list[CheckResult] = []
    checks.extend(
        require_file(
            "mobile/src/screens/LearnScreen.tsx",
            (
                "getLearningDashboard",
                "setLoadError",
                "Retry",
                "Start test",
                "startSession('quick_drill')",
                "LessonSession",
                "Placement",
                "Scenarios",
                "RealTalkHub",
                "learn-find-tutors",
            ),
        )
    )
    checks.extend(
        require_file(
            "frontend/src/pages/Learn.tsx",
            ("getDashboard", "startSession('quick_drill')", "/learn/placement", "/learn/scenarios", "/learn/session"),
        )
    )
    checks.extend(
        require_file(
            "mobile/src/screens/ScenarioRoleplayScreen.tsx",
            ("startScenario", "sendScenarioMessage", "Sparky is typing", "requestScenarioHint"),
        )
    )
    checks.extend(
        require_file(
            "mobile/src/screens/ChatScreen.tsx",
            ("sparky-input", "sparky-send", "sparky-loading", "sparky-assistant-message", "grammarLearn", "sparkyError"),
        )
    )
    checks.extend(
        require_file(
            "mobile/src/screens/__tests__/learning.test.tsx",
            ("Quick Drills", "PlacementScreen", "Sparky is typing", "LessonSessionScreen"),
        )
    )

    scenario = REPO_ROOT / "mobile/src/screens/ScenarioRoleplayScreen.tsx"
    if scenario.exists():
        text = scenario.read_text(encoding="utf-8", errors="replace")
        if re.search(r"startScenario[\s\S]*\.catch\(\(\)\s*=>\s*\{\s*\}\)", text):
            checks.append(
                fail_check(
                    "Scenario roleplay start errors are visible",
                    "startScenario currently has a silent catch; learner-facing failure must show retry/error UI",
                )
            )
        else:
            checks.append(pass_check("Scenario roleplay start errors are visible"))

    return GateResult("learning dashboard contract", checks)


def run_wireframe_parity_gate() -> GateResult:
    if not shutil.which("bash"):
        return GateResult("wireframe parity", [fail_check("bash available for verify-wireframe-parity.sh")])
    return GateResult(
        "wireframe parity",
        [run_command("verify-wireframe-parity.sh", "bash deploy/ci/verify-wireframe-parity.sh", timeout=300)],
    )


def role_commands(role: str) -> list[tuple[str, str]]:
    role = role or ""
    commands: dict[str, list[tuple[str, str]]] = {
        "backend_engineer": [
            ("backend build", "cd backend && go build ./..."),
            ("backend vet", "cd backend && go vet ./..."),
            ("backend tests", "cd backend && go test ./..."),
        ],
        "frontend_engineer": [
            ("frontend tests", "cd frontend && npm test"),
            ("frontend build", "cd frontend && npm run build"),
        ],
        "mobile_engineer": [
            ("mobile typecheck", "cd mobile && npx tsc --noEmit"),
            ("mobile tests", "cd mobile && npm test"),
        ],
        "test_engineer": [
            ("frontend tests", "cd frontend && npm test"),
            ("mobile tests", "cd mobile && npm test"),
        ],
        "sre": [
            ("prod compose config", "docker compose -f docker-compose.prod.yml config --quiet"),
        ],
        "qa_engineer": [],
        "reviewer": [],
        "analyst": [],
        "product_manager": [],
        "teacher": [],
    }
    return commands.get(role, [])


def run_task_gate(task_id: str, task_name: str, role: str) -> GateResult:
    checks: list[CheckResult] = []
    lower = f"{task_id} {task_name} {role}".lower()

    if any(k in lower for k in ("qa", "test", "acceptance", "tdd", "release", "gate", "wireframe")):
        checks.extend(run_test_quality_contract().checks)
    if any(k in lower for k in ("learn", "learning", "daily", "drill", "placement", "sparky", "scenario", "ai tutor")):
        checks.extend(run_learning_contract().checks)
    if any(k in lower for k in ("wireframe", "parity", "release", "gate")):
        checks.extend(run_wireframe_parity_gate().checks)
    if any(k in lower for k in ("release", "gate", "acceptance", "tdd rescue", "final")):
        append_command_checks(checks, release_acceptance_commands())

    for name, command in role_commands(role):
        checks.append(run_command(name, command))

    if not checks:
        checks.append(pass_check("no deterministic command required for this role", "state still depends on phase/full gates"))

    return GateResult(f"task gate {task_id}", checks)


def run_full_gate(run_commands: bool = True) -> GateResult:
    checks: list[CheckResult] = []
    checks.extend(run_static_contract_gate().checks)
    if run_commands:
        for role in ("backend_engineer", "frontend_engineer", "mobile_engineer"):
            append_command_checks(checks, role_commands(role))
        append_command_checks(checks, release_acceptance_commands())
    return GateResult("full release gate", checks)
