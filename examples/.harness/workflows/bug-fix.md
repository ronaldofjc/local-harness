# Bug Fix Workflow

## Overview
Use this workflow for diagnosing and fixing bugs with minimal regression risk.

## Phases

### 1. Investigate
- Reproduce the bug consistently.
- Gather logs, stack traces, and error messages.
- Identify the scope of impact.

### 2. Reproduce
- Create a minimal reproduction case.
- Add a failing test that demonstrates the bug.
- Confirm the test fails before the fix.

### 3. Fix
- Make the smallest possible change to fix the root cause.
- Avoid fixing symptoms only.
- Run sensors after the change (`go test`, `govet`, `staticcheck`).

### 4. Spec / Regression
- Ensure the failing test now passes.
- Add edge-case tests if applicable.
- Run `contract.spec.validate` if a spec covers this area.

### 5. Complete
- Update documentation if the behaviour changed.
- Record evidence in the task via `contract.task.complete`.
- Log the session via `session.append` for future steering analysis.

## Exit Criteria
- Bug is reproducible via test before fix and passing after fix.
- No new lint or test failures introduced.
- Evidence recorded and task completed.
