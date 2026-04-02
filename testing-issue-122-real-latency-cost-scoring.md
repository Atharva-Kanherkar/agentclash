# Issue #122 Locked Plan and Merge Gate

Branch: `issue-122-real-latency-cost-scoring`

Status: LOCKED

This branch is reserved for issue `#122`: implementing real latency and cost metric collection in scoring dimensions.

Only the work listed in this document is in scope for the PR that comes from this branch. If new ideas come up while implementing, they must be deferred to follow-up issues or follow-up PRs unless they are strictly required to complete one of the locked items below.

## Locked Scope

The PR is allowed to do only these things:

1. Add challenge-pack manifest support for runtime limits and latency/cost normalization inputs needed by scoring.
2. Add configurable pricing support keyed by provider and model.
3. Compute latency from persisted run events instead of returning stub warnings.
4. Compute token-based model cost from persisted run events instead of returning stub warnings.
5. Normalize latency and cost into real scorecard dimension scores.
6. Preserve provider-neutral scoring design even if the initial pricing table only includes OpenAI models.
7. Add or update unit tests for manifest loading, pricing lookup, metric aggregation, normalization, and unavailable-data behavior.
8. Update docs only where necessary to reflect the new manifest/config contract.

## Explicitly Out of Scope

The PR must not expand into these areas:

1. Adding new model providers beyond what already exists.
2. Building provider-specific scoring branches inside the scoring engine.
3. Charging separate tool-execution or sandbox-infrastructure cost as part of this issue.
4. Building dashboards, analytics screens, or unrelated observability UI.
5. Refactoring unrelated scoring paths or repository flows.
6. Introducing hardcoded benchmark budgets in Go code when they belong in challenge-pack configuration.
7. Automatic model-price fetching from third-party services.

## Non-Negotiable Design Rules

1. Challenge packs define benchmark expectations and limits.
2. Provider adapters emit canonical telemetry.
3. Scoring consumes canonical telemetry and config; it does not parse provider-specific billing rules directly.
4. Pricing must be configurable, not embedded as permanent scoring constants.
5. Unknown pricing must degrade to unavailable cost scoring, not fake values.
6. OpenAI-only initial pricing coverage is acceptable, but the architecture must remain provider-neutral.

## Merge Gate

This PR must not be merged unless all of the following are true:

1. Latency score is produced from real persisted run-event timing evidence.
2. Cost score is produced from real persisted token-usage evidence plus pricing config.
3. Challenge-pack-configured limits or normalization inputs are wired into scoring.
4. Stub warnings for latency and cost are removed or replaced by real unavailable-data reasons.
5. Missing pricing and missing usage cases are covered by tests.
6. Unit tests covering the new logic pass locally.
7. The PR stays within the locked scope in this file.

## Review Standard

If the implementation starts drifting into provider expansion, infra-cost accounting, or unrelated scoring cleanup, the PR should be stopped and reduced back to the scope above before merge.
