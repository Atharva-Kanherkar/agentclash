# codex/issue-322-promotion-endpoint — Test Contract

## Functional Behavior
The promotion endpoint should convert a promotable failure review item into a regression case and audit record in one transactional flow.
- `POST /workspaces/{workspaceID}/runs/{runID}/failures/{challengeIdentityID}/promote` accepts a JSON body with `suite_id`, `promotion_mode`, `title`, and optional `failure_summary`, `severity`, `validator_overrides`, and `metadata`.
- The endpoint returns the created regression case using the existing regression case response schema.
- If the same failure is promoted again for the same suite and run agent, the endpoint returns the existing case with `200 OK` and does not create a second promotion audit row.
- Promotion is allowed only when the failure exists, is `promotable == true`, the requested `promotion_mode` is listed in `promotion_mode_available`, the suite belongs to the same workspace, the suite is `active`, and the suite's source pack matches the run's challenge pack.
- Severity defaults to `blocking` for `policy_violation` and `sandbox_failure` failures when omitted; all other omitted severities default to `warning`.
- `validator_overrides` only accepts `judge_threshold_overrides` and `assertion_toggles`; any other top-level key is rejected with `400`.
- The created regression case freezes the challenge payload snapshot from the run execution context and a narrowed expected contract from the source evaluation spec.

## Unit Tests
- `TestDefaultPromotionSeverity` verifies explicit severity wins and implicit severity follows the policy/sandbox vs warning rule.
- `TestValidatePromotionOverrides` rejects unknown keys, accepts allowed keys, and tolerates null or omitted overrides.
- `TestParsePromoteFailureRequest` rejects malformed or missing required request fields.

## Integration / Functional Tests
- Regression manager promotion tests cover invalid preconditions: missing/non-promotable failure, wrong mode, suite cross-workspace, suite archived, and source pack mismatch.
- Repository-backed promotion flow tests verify transaction behavior: created case fields, promotion audit row, payload snapshot freezing, expected contract subset freezing, and idempotent second POST.
- API endpoint tests verify status codes and wire payloads for happy path, idempotent replay, and validation failures.

## Smoke Tests
- `go test ./backend/internal/domain ./backend/internal/api ./backend/internal/repository`
- `go test ./backend/...` if the focused package suite passes and runtime permits.

## E2E Tests
N/A — this backend issue does not add a browser or CLI user journey.

## Manual / cURL Tests
```bash
curl -X POST http://localhost:8080/v1/workspaces/$WORKSPACE_ID/runs/$RUN_ID/failures/$CHALLENGE_ID/promote \
  -H "Content-Type: application/json" \
  -H "X-User-ID: $USER_ID" \
  -H "X-Workspace-Memberships: $WORKSPACE_ID:workspace_admin" \
  -d '{
    "suite_id":"'$SUITE_ID'",
    "promotion_mode":"full_executable",
    "title":"Filesystem policy regression",
    "failure_summary":"Policy guard tripped",
    "validator_overrides":{"judge_threshold_overrides":{"policy.filesystem":1}},
    "metadata":{"source":"manual-check"}
  }'
# Expected: 201 with regression case JSON including frozen payload_snapshot and expected_contract.

curl -X POST http://localhost:8080/v1/workspaces/$WORKSPACE_ID/runs/$RUN_ID/failures/$CHALLENGE_ID/promote \
  -H "Content-Type: application/json" \
  -H "X-User-ID: $USER_ID" \
  -H "X-Workspace-Memberships: $WORKSPACE_ID:workspace_admin" \
  -d '{
    "suite_id":"'$SUITE_ID'",
    "promotion_mode":"full_executable",
    "title":"Filesystem policy regression"
  }'
# Expected: 200 with the same regression case id as the first request and no duplicate audit row.
```
