# codex/issue-319-failure-review — Test Contract

## Functional Behavior
- `GET /v1/workspaces/{workspaceID}/runs/{runID}/failures` returns a paginated list of normalized `FailureReviewItem` objects for the selected run.
- The handler loads the run first, returns `404` when the run does not exist, and also returns `404` when `run.workspace_id != workspaceID` before workspace authorization is attempted.
- Authorized callers can filter the failure list by `agent_id`, `severity`, `failure_class`, `evidence_tier`, `challenge_key`, and `case_key`.
- Pagination is cursor-based and stable. Results are ordered deterministically so repeating the same request returns the same page and cursor boundary.
- `limit` defaults to `50` and is capped at `200`. Invalid UUIDs, unsupported enums, malformed cursors, or non-positive limits return `400`.
- Each item freezes the wire field names from issue `#319` / parent `#318`:
  `run_id, run_agent_id, challenge_identity_id, challenge_key, case_key, item_key, failure_state, failed_dimensions, failed_checks, failure_class, headline, detail, recommended_action, promotable, promotion_mode_available, replay_step_refs, artifact_refs, judge_refs, metric_refs, evidence_tier`.
- Failure-class inference is isolated in `backend/internal/failurereview/classify.go` and covers the initial taxonomy, including policy, tool-argument, tool-selection, timeout/budget, sandbox, malformed output, incorrect final output, insufficient evidence, flaky placeholder, and fallback `other`.
- Eligibility is computed on read:
  `promotable = run.status == completed AND challenge_identity_id != nil AND evidence_tier != 'none'`.
- `promotion_mode_available` includes `full_executable` only when `evidence_tier` is `native_structured` or `hosted_structured` and the source challenge-pack version is still runnable in the current schema.
- `promotion_mode_available` includes `output_only` only when a final output can be deep-linked from `run_events`.
- Read-model assembly joins existing scoring rows, LLM judge rows, frozen execution context, lightweight replay refs from `run_events`, and optional baseline comparison cues from `run_comparisons` without introducing any new tables.

## Unit Tests
- `TestClassifyFailureClassMappings` covers every initial taxonomy branch.
- `TestPromotionEligibilityByEvidenceTierAndRunStatus` covers run status, challenge identity presence, runnable-pack gating, and final-output availability.
- `TestAssembleFailureReviewItemBuildsRefsAndFailedChecks` verifies the read model maps validator/judge/metric evidence into the expected item shape.

## Integration / Functional Tests
- Repository integration test builds failure-review rows from existing tables only and verifies filtering by run, agent, challenge, case, class, and evidence tier.
- Repository integration test verifies cursor pagination remains stable across repeated reads and honors the `limit` cap.
- Repository integration test includes optional comparison data and confirms baseline cues are omitted when no comparison exists.

## Smoke Tests
- `go test ./backend/internal/failurereview ./backend/internal/api ./backend/internal/repository -short -race -count=1`
- `cd backend && sqlc generate`
- `npx @redocly/cli lint docs/api-server/openapi.yaml`

## E2E Tests
- N/A — not applicable for this backend/API-only change.

## Manual / cURL Tests
```bash
curl -sS \
  -H "X-Dev-User-Id: <user-id>" \
  -H "X-Dev-Workspace-Memberships: <workspace-id>:workspace_member" \
  "http://localhost:8080/v1/workspaces/<workspace-id>/runs/<run-id>/failures"
# Expected: 200 with {"items":[...],"next_cursor":...}
```

```bash
curl -sS \
  -H "X-Dev-User-Id: <user-id>" \
  -H "X-Dev-Workspace-Memberships: <workspace-id>:workspace_member" \
  "http://localhost:8080/v1/workspaces/<workspace-id>/runs/<run-id>/failures?failure_class=tool_argument_error&evidence_tier=native_structured&limit=1"
# Expected: 200 with filtered items and a stable next_cursor when more rows exist.
```

```bash
curl -sS \
  -H "X-Dev-User-Id: <user-id>" \
  -H "X-Dev-Workspace-Memberships: <workspace-id>:workspace_member" \
  "http://localhost:8080/v1/workspaces/<other-workspace-id>/runs/<run-id>/failures"
# Expected: 404 when the run belongs to a different workspace.
```
