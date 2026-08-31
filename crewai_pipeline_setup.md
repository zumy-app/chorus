# System Instruction: Autonomous CrewAI Multi-Agent Pipeline Setup

**Context:**
We are setting up a fully autonomous multi-agent software engineering loop using **CrewAI** inside our local repository: **`chorus`** (React Native/React frontend, Go backend, PostgreSQL, Redis). The objective is to replace the human manager completely. The system must ingest requirements, modify files, run the existing integration test runners, handle compilation errors, and iterate until the test suite passes flawlessly.

---

## 🚀 Step 1: Environment & Dependency Management
First, initialize the isolation layer. Create a dedicated tools requirements list and install them in the workspace environment.

1. Create a `requirements-agents.txt` file at the root containing:
```text
crewai>=0.30.0
crewai-tools>=0.0.15
pydantic>=2.0.0
```

2. Execute the installation script in the terminal:
```bash
pip install -r requirements-agents.txt
```

---

## 🛠️ Step 2: Create Custom Local Terminal Tool
To allow the QA agent to evaluate code changes without breaking window persistence, create a specialized execution class. Create `tools/terminal_tool.py`:

```python
import subprocess
from crewai.tools import BaseTool
from pydantic import BaseModel, Field

class TerminalExecutionSchema(BaseModel):
    command: str = Field(..., description="The exact bash/shell command to execute inside the workspace directory.")

class LocalTerminalTool(BaseTool):
    name: str = "Local Terminal Runner"
    description: str = "Executes arbitrary terminal commands locally and returns the exact stdout and stderr string output. Essential for running test suites and checking compilation."
    args_schema: type[BaseModel] = TerminalExecutionSchema

    def _run(self, command: str) -> str:
        try:
            # Execute command with a timeout to catch hanging servers or infinite loops safely
            result = subprocess.run(
                command,
                shell=True,
                capture_output=True,
                text=True,
                timeout=120
            )
            output = f"STDOUT:\n{result.stdout}\nSTDERR:\n{result.stderr}\nEXIT CODE: {result.returncode}"
            return output
        except subprocess.TimeoutExpired:
            return "ERROR: Command execution timed out after 120 seconds."
        except Exception as e:
            return f"ERROR: Execution failed: {str(e)}"
```

---

## 🤖 Step 3: Implement the Core Autonomous Orchestrator Loop
Create the main controller file at the root: `orchestrator.py`. This script maps the software development lifecycle directly into an algorithmic state loop.

```python
import os
from crewai import Agent, Crew, Process, Task
from crewai.tools import FileReadTool, FileWriterTool, DirectoryReadTool
from tools.terminal_tool import LocalTerminalTool

# Initialize core environmental access tools
file_reader = FileReadTool()
file_writer = FileWriterTool()
dir_reader = DirectoryReadTool()
terminal_runner = LocalTerminalTool()

# =====================================================================
# AGENT DEFINITIONS
# =====================================================================

product_manager = Agent(
    role="Product Manager",
    goal="Analyze initial requirements from REQUIREMENTS.md and build structural implementation milestones.",
    backstory="You are a strict technical Product Manager. You map functional gaps by cross-referencing user specifications against the current codebase structure.",
    verbose=True,
    tools=[file_reader, dir_reader],
    memory=True
)

lead_developer = Agent(
    role="Lead Software Engineer",
    goal="Implement multi-file architecture code changes covering Go APIs, Redis registries, and React/React Native structures.",
    backstory="You are an expert full-stack engineer. You read filesystem states, alter schemas, write Go micro-services, and design unified UI components seamlessly.",
    verbose=True,
    tools=[file_reader, file_writer, dir_reader],
    memory=True
)

qa_engineer = Agent(
    role="Quality Assurance Engineer",
    goal="Validate stability by executing test scripts and routing stderr messages back to development if validation parameters fail.",
    backstory="You are a zero-tolerance QA engineer. You execute local scripts, catch error codes or logs, and refuse to accept task completion until all validation sweeps return successfully.",
    verbose=True,
    tools=[terminal_runner],
    allow_delegation=True  # Allows QA to explicitly tell the developer what broke and command a rebuild
)

# =====================================================================
# TASK ROUTING & CONTINUOUS LOOP
# =====================================================================

task_analyze_backlog = Task(
    description=(
        "1. Read the comprehensive specification logs inside REQUIREMENTS.md and BACKLOG_REFINEMENT_2026-08-23.md.\n"
        "2. Analyze the current folder structure in the workspace to discover which backend-available endpoints "
        "(Presence, Grammar analysis, Vocabulary modules) are completely missing from the current React/Mobile views.\n"
        "3. Produce a structured execution document outlining exactly what modules need code modifications."
    ),
    expected_output="A prioritized markdown file detailing explicit multi-file source adjustments.",
    agent=product_manager,
    output_file="current_sprint_backlog.md"
)

task_execute_code = Task(
    description=(
        "1. Read the newly generated 'current_sprint_backlog.md'.\n"
        "2. Systematically perform code edits across the repository.\n"
        "3. Ensure that Go modules, PostgreSQL migration files, and React TypeScript files match the data flows exactly."
    ),
    expected_output="Successful file modifications written directly to the workspace.",
    agent=lead_developer,
)

task_verify_and_heal = Task(
    description=(
        "1. Execute the project test commands using the Local Terminal Runner tool:\n"
        "   - Run Go backend tests: 'cd backend && go test ./...'\n"
        "   - Run Frontend test suite: 'cd frontend && npm test'\n"
        "   - Run E2E specs: 'cd e2e && npx playwright test'\n"
        "2. Evaluate the precise string results of STDOUT and STDERR.\n"
        "3. CRITICAL: If compilation fails, database migrations stall, or tests return an exit code other than 0, "
        "delegate a task back to the Lead Software Engineer with the exact error log block and command a bugfix refactor.\n"
        "4. Re-run the tests continuously until everything finishes with zero validation errors."
    ),
    expected_output="Comprehensive validation confirming all tests pass perfectly without any error output.",
    agent=qa_engineer
)

# Initialize the automated execution engine
development_crew = Crew(
    agents=[product_manager, lead_developer, qa_engineer],
    tasks=[task_analyze_backlog, task_execute_code, task_verify_and_heal],
    process=Process.sequential,
    verbose=True
)

if __name__ == "__main__":
    print("🤖 Starting the Autonomous Engineering Lifecycle Loop for Chorus App...")
    result = development_crew.kickoff()
    print("\n🏁 Target state achieved! System verification complete.")
```

---

## 🏁 Step 4: Kickoff Action
OpenCode agent, write all files listed above to their designated relative paths, resolve the package dependencies, and run:
```bash
python orchestrator.py
```
Let the automated engine loops execute. The multi-agent pipeline will take over from here, running continuous code refinement cycles overnight until all requirements match a completely clean integration test pass.