# codex/issue-321-regression-suite-crud — Test Contract

## Functional Behavior
- Add goose migration `backend/db/migrations/00023_regression_suites.sql` that creates `workspace_regression_suites`, `workspace_regression_cases`, and `workspace_regression_promotions` with the foreign keys, `CHECK` constraints, defaults, and indexes described in issue #321.
- Enforce `UNIQUE (workspace_id, name)` only for active suites, and drop objects in reverse dependency order in the down migration.
- Add `backend/internal/domain/regression.go` with typed string enums for suite status, case status, severity, and promotion mode.
- Allow suite transitions `active -> archived` and `archived -> active`.
- Allow case transitions `active -> muted`, `active -> archived`, `muted -> active`, and `muted -> archived`; reject transitions from `archived`.
- Mirror transition enforcement in repository update paths with `WHERE status = @from_status`, returning the repository invalid-transition error on stale or illegal transitions.
- Add sqlc-backed CRUD for suites and cases plus promotion insertion primitives so later promotion flow work can reuse them.
- Add manager logic that always receives `workspaceID`, loads the target resource, and returns the existing `ErrForbidden` / `ErrNotFound` style errors for cross-workspace access and missing records.
- Add HTTP endpoints for create/list/get/patch suite operations, list suite cases, and patch case operations under `/v1`.
- Pagination for `GET /workspaces/{ws}/regression-suites` should follow existing workspace list patterns and remain scoped to the requested workspace.
- Patch endpoints must only update the supported fields from the issue: suite `name`, `description`, `status`, `default_gate_severity`; case `status`, `severity`, `title`, `description`.
- Add OpenAPI schemas and paths for regression suites, cases, promotions, and their create/patch inputs in `docs/api-server/openapi.yaml`, then lint successfully.

## Unit Tests
- `backend/internal/domain/regression_test.go`
  - suite transition matrix accepts only `active <-> archived`
  - case transition matrix accepts only the allowed active/muted/archived flows
  - invalid raw suite status, case status, severity, and promotion mode values are rejected
- Repository transition coverage in `backend/internal/repository/...`
  - patching a suite with an invalid status transition returns `repository.ErrInvalidTransition`
  - patching a case with an invalid status transition returns `repository.ErrInvalidTransition`

## Integration / Functional Tests
- Repository integration test against Postgres covers:
  - create suite -> get suite -> list suites by workspace -> patch suite
  - create case -> get case -> list cases by suite -> patch case
  - promotion row insert succeeds through the repository/sqlc path
  - suite and case transition guards behave correctly against the real database
- Manager tests cover:
  - cross-workspace suite fetch/update/list access is denied
  - cross-workspace case patch access is denied
- Handler tests cover:
  - `POST /workspaces/{ws}/regression-suites` create succeeds
  - `GET /workspaces/{ws}/regression-suites/{suiteID}` returns the created suite
  - `PATCH /workspaces/{ws}/regression-suites/{suiteID}` updates allowed fields
  - `GET /workspaces/{ws}/regression-suites` lists the suite with pagination inputs
  - `GET /workspaces/{ws}/regression-suites/{suiteID}/cases` returns suite cases
  - `PATCH /workspaces/{ws}/regression-cases/{caseID}` updates allowed fields

## Smoke Tests
- `cd backend && go test ./internal/domain ./internal/repository ./internal/api`
- `cd backend && sqlc generate`
- `npx @redocly/cli lint docs/api-server/openapi.yaml`

## E2E Tests
- N/A — backend CRUD change only; no browser or full user-journey automation is required in this issue.

## Manual / cURL Tests
```bash
curl -X POST http://localhost:8080/v1/workspaces/$WORKSPACE_ID/regression-suites \
  -H "Content-Type: application/json" \
  -H "X-Dev-User-ID: $USER_ID" \
  -d '{
    "source_challenge_pack_id": "'"$CHALLENGE_PACK_ID"'",
    "name": "Critical regressions",
    "description": "Cases promoted from production failures",
    "default_gate_severity": "warning"
  }'
# Expected: 201 with a RegressionSuite response.

curl -X PATCH http://localhost:8080/v1/workspaces/$WORKSPACE_ID/regression-suites/$SUITE_ID \
  -H "Content-Type: application/json" \
  -H "X-Dev-User-ID: $USER_ID" \
  -d '{"status":"archived","default_gate_severity":"blocking"}'
# Expected: 200 with updated suite fields.

curl -X PATCH http://localhost:8080/v1/workspaces/$WORKSPACE_ID/regression-cases/$CASE_ID \
  -H "Content-Type: application/json" \
  -H "X-Dev-User-ID: $USER_ID" \
  -d '{"status":"muted","severity":"warning","title":"Mute flaky case"}'
# Expected: 200 with updated case fields.
```
