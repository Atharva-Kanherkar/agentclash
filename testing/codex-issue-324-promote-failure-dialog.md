# codex/issue-324-promote-failure-dialog - Test Contract

## Functional Behavior
- The Failures page shows a `Promote` action for promotable failure rows and hides it for non-promotable rows.
- Clicking `Promote` opens a dialog prefilled from the selected `FailureReviewItem`:
  - Title defaults to `headline` and remains editable.
  - Failure summary defaults to `detail` and remains editable.
  - Promotion mode is limited to `promotion_mode_available[]` and defaults to `full_executable` when present.
  - Severity defaults to the client-mirrored server rule: `blocking` for `policy_violation` and `sandbox_failure`, otherwise `warning`.
- Destination suite options are limited to regression suites in the same workspace whose `source_challenge_pack_id` matches the failure's source pack.
- If no eligible suites exist, the dialog shows an inline empty state with a `Create suite` link that routes to the regression suites page and pre-opens the create-suite modal with the failure's source pack selected.
- Advanced validator overrides stay collapsed by default and only expose judge-threshold number inputs plus assertion-toggle checkboxes; no free-form JSON editor is shown.
- Submitting the dialog calls the promote-failure API with the edited values and normalized validator overrides.
- A newly created promotion shows a success toast and links to the new regression case detail page.
- An idempotent success path shows `Already promoted - open case` instead of a green success toast when the server reports an existing case.
- API errors surface the server message verbatim in an error toast.
- This branch intentionally excludes the run-creation regression-suite selector because `regression_suite_ids[]` / `regression_case_ids[]` are not available on `origin/main` as of 2026-04-19.

## Unit Tests
- `web/src/lib/api/__tests__/regression.test.ts`
  - `listRegressionSuites` sends workspace-scoped list requests with pagination params.
  - `promoteFailure` posts the expected payload shape.
  - `buildPromotionOverrides` omits empty groups and serializes non-empty threshold/toggle overrides.
  - `defaultPromotionSeverityForFailure` mirrors the backend default-severity rules for failure classes.

## Integration / Functional Tests
- `pnpm lint` in `web/` passes after the dialog wiring and route/query updates.
- `npx tsc --noEmit` in `web/` passes with the new API types and client component state.

## Smoke Tests
- Open a run failures page with promotable items and confirm the list still renders, filters still work, and the detail drawer still opens.
- Open the Promote dialog, cancel it, and confirm the Failures page remains usable without stale loading state.

## E2E Tests
- N/A - no automated browser E2E coverage is added in this branch.

## Manual / cURL Tests
```bash
# Promote a failure into an existing matching suite.
curl -X POST "$API_URL/v1/workspaces/$WORKSPACE_ID/runs/$RUN_ID/failures/$CHALLENGE_ID/promote" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "suite_id": "'"$SUITE_ID"'",
    "promotion_mode": "full_executable",
    "title": "Filesystem policy regression",
    "failure_summary": "Model attempted a forbidden file write.",
    "severity": "blocking",
    "validator_overrides": {
      "judge_threshold_overrides": { "policy.filesystem": 0.9 },
      "assertion_toggles": { "capture.files": true }
    }
  }'
# Expected: 201 for a new case or 200 for an idempotent replay, response body includes `case.id`.
```

```text
Manual browser flow:
1. Open /workspaces/{workspaceId}/runs/{runId}/failures.
2. Click Promote on a promotable row and verify prefilled title, summary, mode, and severity.
3. Promote into a matching suite and use the toast CTA to open /workspaces/{workspaceId}/regression-suites/{suiteId}/cases/{caseId}.
4. Re-submit the same promotion and verify the UI shows the idempotent "Already promoted - open case" path.
5. Use a workspace with no matching suites, click Create suite, and verify the regression suites page opens with the create modal already open and the source pack preselected.
```
