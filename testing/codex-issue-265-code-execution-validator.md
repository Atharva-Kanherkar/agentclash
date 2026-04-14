# codex/issue-265-code-execution-validator — Test Contract

## Functional Behavior
Add a deterministic `code_execution` validator that runs a configured test command inside the existing live sandbox before teardown and scores the outcome from the command result.

- A validator with `"type": "code_execution"` is accepted by spec loading and validation when:
- `target` is a `file:` reference that maps to a declared post-execution file capture key.
- `config.test_command` is non-empty.
- `config.timeout_ms` is greater than 0 when provided.
- `config.scoring` is one of `fraction_passed` or `all_or_nothing`.
- `config.pass_threshold` is between 0 and 1 when provided.
- Native executor runs `code_execution` verification before normal sandbox teardown so tests execute against the real generated workspace.
- Verification emits structured replay evidence that includes command, exit code, stdout, stderr, timeout status, parsed test counts when available, and the computed validator score inputs.
- Scoring resolves `code_execution` validator results from that verification evidence instead of captured file contents.
- `fraction_passed` returns `passed_tests / total_tests` when counts are available, otherwise falls back to `1` on exit code 0 and `0` on non-zero exit code.
- `all_or_nothing` returns `1` only when all discovered tests pass or the command exits 0 with no parsed failures; otherwise `0`.
- Timeouts are surfaced as a failed validator with a score of `0` and a timeout reason.
- Packs that do not declare `code_execution` behave exactly as before.
- `pass_at_k` is rejected during validation in this implementation because the current run-agent architecture executes a single sample, not a multi-sample pass@k batch.

## Unit Tests
- `TestValidateEvaluationSpec_AcceptsCodeExecutionValidator` — valid `code_execution` config passes validation.
- `TestValidateEvaluationSpec_RejectsPassAtKCodeExecution` — `pass_at_k` is rejected with a clear validation error.
- `TestExecuteCodeExecutionCheck_ParsesPytestCounts` — executor captures pass/fail counts from test output and includes them in payload.
- `TestExecuteCodeExecutionCheck_Timeout` — timed-out command records timeout, non-passing status, and zero score input.
- `TestEvaluateRunAgent_CodeExecutionFractionPassed` — scoring produces a graduated score from verification evidence.
- `TestEvaluateRunAgent_CodeExecutionAllOrNothing` — scoring returns binary pass/fail from verification evidence.

## Integration / Functional Tests
- Native executor with a fake sandbox session runs a configured `test_command` before teardown and observer records a `grader.verification` event for it.
- End-to-end scoring path consumes the recorded verification event and persists a `code_execution` validator result without changing legacy validator behavior.

## Smoke Tests
- `go test ./backend/internal/scoring ./backend/internal/engine ./backend/internal/worker`
- Existing post-execution file capture tests still pass, confirming backward compatibility.

## E2E Tests
N/A — this change is backend-only and is covered by package-level integration tests around the native executor and scoring pipeline.

## Manual / cURL Tests
N/A — there is no stable local API surface for manually invoking this validator without constructing a full run fixture. Reviewer verification is via targeted Go tests for executor + scoring behavior.
