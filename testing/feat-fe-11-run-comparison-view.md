# feat/fe-11-run-comparison-view — Test Contract

## Functional Behavior

### Comparison Page (`/workspaces/{id}/compare?baseline={runId}&candidate={runId}`)
- Header shows baseline run name vs candidate run name
- Comparison state displayed: `comparable`, `partial_evidence`, `not_comparable`
- When `not_comparable`: reason is shown clearly, deltas are not rendered
- When `partial_evidence`: a warning banner is shown about incomplete data
- When `comparable`: full comparison renders with deltas
- Summary text from backend displayed
- Key deltas table:
  - Columns: Metric name, Baseline value, Candidate value, Delta (+/-)
  - Green color for improvements (outcome = "better")
  - Red color for regressions (outcome = "worse")
  - Neutral for "same" or "unknown"
- Regression reasons list shown when present
- Evidence quality warnings shown when present
- Missing query params → empty state with instructions

### Compare Launcher — Run List
- Checkbox column in run list table for selecting runs
- "Compare" button appears when exactly 2 runs are selected
- Button navigates to `/workspaces/{id}/compare?baseline={runId1}&candidate={runId2}`
- Selection resets on page navigation

### Compare Launcher — Run Detail
- "Compare with..." button in run detail header
- Opens a dialog with a list of other runs in the workspace
- Selecting a run navigates to comparison page with current run as baseline

### Fallback
- Backend serves HTML at `GET /v1/compare/viewer` — can iframe as fallback
- N/A for initial implementation — native comparison view is the primary target

## Unit Tests
N/A — frontend components; covered by type checking and integration tests.

## Integration / Functional Tests
- TypeScript compiles without errors: `cd web && npx tsc --noEmit`
- ESLint passes: `cd web && pnpm lint`
- Production build succeeds: `cd web && pnpm build`

## Smoke Tests
- Dev server starts without errors: `cd web && pnpm dev`
- Navigate to `/workspaces/{id}/runs` — run list renders with checkbox column
- Select 2 runs — Compare button appears
- Navigate to `/workspaces/{id}/compare?baseline=X&candidate=Y` — page loads

## E2E Tests
N/A — no E2E test framework in place for frontend.

## Manual / cURL Tests

### 1. Compare page renders with valid parameters
Navigate to: `http://localhost:3000/workspaces/{workspaceId}/compare?baseline={runId1}&candidate={runId2}`
- Expected: comparison page loads, shows header with run names, delta table, state indicator

### 2. Compare page handles missing parameters
Navigate to: `http://localhost:3000/workspaces/{workspaceId}/compare`
- Expected: empty state telling user to select two runs to compare

### 3. Run list compare flow
Navigate to: `http://localhost:3000/workspaces/{workspaceId}/runs`
- Check two runs using checkboxes
- Click "Compare" button
- Expected: navigated to compare page with the two run IDs as query params

### 4. Run detail compare flow
Navigate to: `http://localhost:3000/workspaces/{workspaceId}/runs/{runId}`
- Click "Compare with..." button
- Select a run from the dialog
- Expected: navigated to compare page with current run as baseline

### 5. Backend API integration
```bash
curl -s http://localhost:8080/v1/compare\?baseline_run_id=X\&candidate_run_id=Y \
  -H "Authorization: Bearer $TOKEN" | jq .state
# Expected: "comparable", "partial_evidence", or "not_comparable"
```
