# codex/issue-325-run-regression-selection — Test Contract

## Functional Behavior
- `POST /workspaces/{workspaceID}/runs` accepts optional `regression_suite_ids` and `regression_case_ids` alongside a challenge pack selection.
- The run manager rejects suites or cases that do not belong to the target workspace.
- The run manager rejects regression cases whose `source_challenge_pack_id` does not match the run's challenge pack.
- The run manager de-duplicates case identities when the same case is referenced directly, through one or more suites, or through the official pack.
- `official_pack_mode` defaults to `"full"` and includes the official pack case set in the resolved run selection.
- `official_pack_mode = "suite_only"` excludes official-only pack cases and uses only the resolved regression selection.
- Run persistence stores the resolved case selection together with the source metadata needed for workflow execution and read-model reporting.
- Workflow execution iterates only over the resolved case selection persisted on the run.
- Scoring results for executed regression-backed cases persist the matching `regression_case_id`; official-only cases leave that field unset.
- Run detail responses expose regression coverage grouped by suite plus any selected regression cases that did not map to suite coverage, and per-case results include `regression_case_id` when present.

## Unit Tests
- Run selection expansion de-duplicates repeated challenge identities across suites, explicit cases, and official pack membership.
- Validation rejects unknown suite IDs, unknown case IDs, workspace mismatches, and challenge-pack mismatches.
- Coverage aggregation reports suite case counts and pass/fail counts from scoring results tagged with `regression_case_id`.
- Request decoding preserves the default `official_pack_mode = "full"` and honors `"suite_only"` when set.

## Integration / Functional Tests
- Creating a run with suite IDs and explicit case IDs persists the resolved selection with correct source annotations.
- Creating a run in `"suite_only"` mode omits official-only pack cases from the stored selection.
- Executing a run uses the persisted selection and only scores the selected challenge identities.
- Scoring persistence writes `regression_case_id` for regression-backed cases and leaves it null for official-only cases.
- Run detail read models include regression coverage and per-case `regression_case_id` data after execution.

## Smoke Tests
- Targeted backend tests for run manager, workflow execution, and run read models pass.
- SQL generation and API type compilation pass after schema and query changes.

## E2E Tests
- N/A — this change is backend-only and does not introduce a separate UI flow in this issue.

## Manual / cURL Tests
```bash
curl -X POST http://localhost:8080/workspaces/$WORKSPACE_ID/runs \
  -H "Content-Type: application/json" \
  -d '{
    "challenge_pack_id": "'"$PACK_ID"'",
    "regression_suite_ids": ["'"$SUITE_ID"'"],
    "regression_case_ids": ["'"$CASE_ID"'"]
  }'
# Expected: 201 response with a run whose persisted selection includes deduplicated
# official and regression-backed challenge identities.

curl -X POST http://localhost:8080/workspaces/$WORKSPACE_ID/runs \
  -H "Content-Type: application/json" \
  -d '{
    "challenge_pack_id": "'"$PACK_ID"'",
    "official_pack_mode": "suite_only",
    "regression_case_ids": ["'"$CASE_ID"'"]
  }'
# Expected: 201 response whose stored selection excludes official-only pack cases.

curl -X POST http://localhost:8080/workspaces/$WORKSPACE_ID/runs \
  -H "Content-Type: application/json" \
  -d '{
    "challenge_pack_id": "'"$PACK_ID"'",
    "regression_suite_ids": ["'"$OTHER_WORKSPACE_SUITE_ID"'"]
  }'
# Expected: 400 or 404 because the suite does not belong to the workspace.
```
