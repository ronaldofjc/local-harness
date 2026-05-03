# PREVC — Plan, Research, Execute, Verify, Commit

## Overview
Use this workflow for implementing new features or tasks with structured phases.

## Phases

### 1. Plan
- Understand the requirement fully.
- Identify the smallest safe change that delivers value.
- List acceptance criteria and risks.

### 2. Research
- Read existing code, tests, and documentation.
- Identify patterns and conventions to follow.
- Look for similar implementations.

### 3. Execute
- Implement the change in small increments.
- Run sensors (`go test`, `staticcheck`, `gofmt`) after each increment.
- Update tests and documentation as needed.

### 4. Verify
- Validate against acceptance criteria.
- Run the full harness: `sensor.run` for all relevant sensors.
- Use `judge.review` if a rubric applies.

### 5. Commit
- Write a clear commit message following Conventional Commits.
- Ensure no secrets or debug code are left behind.
- Mark the task as complete via `contract.task.complete`.

## Exit Criteria
- All sensors pass.
- Judge review is conclusive and passed (if applicable).
- Task evidence is recorded.
