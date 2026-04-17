# codex/issue-300-scorecard-dimensions — Test Contract

## Functional Behavior
The run-agent scorecard response should expose per-dimension scoring metadata inside `scorecard.dimensions`.
- For each available or unavailable dimension already present in the stored scorecard document, preserve the existing fields and add `weight`, `contribution`, `pass_threshold`, `gate`, and `gate_passed`.
- `weight` should come from the evaluation spec's `scorecard.dimensions[*].weight`, defaulting to `1.0` when omitted so the response explains the effective scoring behavior.
- `contribution` should reflect the dimension's weighted contribution to the overall score using the same weighting semantics as the scoring engine.
- For weighted and binary strategies, `contribution` should be `weight * score` for available dimensions and omitted when a dimension score is unavailable.
- For hybrid strategy scorecards, gated dimensions should report `contribution: 0` because they act as hard pass/fail checks and are excluded from the weighted non-gate average; non-gated dimensions should report `weight * score`.
- `pass_threshold` should mirror the dimension declaration threshold when present.
- `gate` should mirror the dimension declaration gate flag.
- `gate_passed` should be:
  - `true` when the dimension is a required gate and its available score meets the threshold
  - `false` when the dimension is a required gate and its score is unavailable or below the threshold
  - omitted when the dimension is not a required gate for the active strategy
- The endpoint should not fabricate entries for dimensions that are absent from the stored scorecard JSON.
- Existing top-level scorecard response fields must remain backward compatible.

## Unit Tests
- `TestEnrichScorecardDimensionsAddsWeightContributionAndGateMetadata` verifies weighted-strategy metadata enrichment.
- `TestEnrichScorecardDimensionsMarksHybridGateContributionAsZero` verifies hybrid gate semantics.
- `TestEnrichScorecardDimensionsHandlesUnavailableRequiredGate` verifies `gate_passed=false` when a required gate has no score.

## Integration / Functional Tests
- `TestGetRunAgentScorecardEndpointReturnsScorecard` verifies the HTTP response includes enriched dimension fields inside `scorecard`.
- OpenAPI schema validation remains aligned with the enriched response shape for `ScorecardDimension`.

## Smoke Tests
- `go test ./backend/internal/api ./backend/internal/scoring`

## E2E Tests
N/A — this change is limited to backend scorecard response shaping and schema documentation.

## Manual / cURL Tests
```bash
curl -s http://localhost:8080/v1/scorecards/$RUN_AGENT_ID \
  -H "X-User-Id: $USER_ID" \
  -H "X-Workspace-Memberships: $WORKSPACE_ID:workspace_member" | jq '.scorecard.dimensions'
# Expected: each returned dimension includes weight/contribution metadata and gate fields when applicable.
```
