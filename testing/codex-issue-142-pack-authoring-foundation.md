# PR Expectations: `codex/issue-142-pack-authoring-foundation`

## Goal

Ship the first end-to-end implementation slice for `#142` based on the locked design:

- workspace-scoped private challenge packs
- YAML-authored pack bundle
- deterministic validators and declared metrics only
- shared validation with field-specific errors
- immutable publish flow
- validate/publish flow without manual DB edits
- forward-compatible asset references in the bundle contract
- no rich UI work in this PR

## Expected Functional Behavior

When this PR is complete:

1. A user can define a challenge-pack bundle in YAML without touching the database directly.
2. The system has a typed pack contract that distinguishes pack metadata, version metadata, challenges, input sets, tool policy, evaluation spec, and asset references.
3. The system validates the pack bundle before publish and returns field-specific errors for invalid authoring input.
4. Publishing a valid bundle creates the necessary persisted pack/version/challenge/input records.
5. Published versions are immutable and new material changes require a new `version_number`.
6. The runtime-facing persisted shape remains compatible with the existing evaluation-spec and run-loading paths.
7. The implementation does not introduce user-authored arbitrary code execution for validators or scorers.
8. Generalized artifact execution remains explicitly deferred if not supported by the current runtime path.

## Expected API / CLI Surface

This PR should provide the minimum surface needed for self-serve authoring and publish flow.

Expected minimum capabilities:

- initialize a new pack authoring skeleton or accept a pack bundle payload
- validate a pack bundle before publish
- publish a pack bundle into persisted runnable data
- read back enough persisted state to confirm publish behavior

If the CLI is not introduced in this PR, the backend/API path must still fully support those operations so a CLI can be layered later without schema redesign.

## Unit Test Expectations

The completed PR should include unit tests for:

- YAML decoding / normalization into the canonical pack model
- validation failures with field-specific paths
- duplicate key/version rejection
- immutability/versioning enforcement
- publish-path persistence mapping from bundle to database records
- rejection of unsupported evaluator/scorer shapes
- forward-compatible asset reference validation

## Integration / Repository Test Expectations

The completed PR should include integration-style coverage for:

- creating or publishing a new pack without manual DB edits
- persisting challenge-pack rows, version rows, challenge rows, and input-set rows from a bundle
- preserving the existing evaluation-spec loading path
- ensuring a published pack can still be loaded by the existing run/evaluation context code

## Manual / Curl Test Expectations

If HTTP endpoints are introduced in this PR, verify with manual requests:

1. Submit an invalid bundle and confirm field-specific validation errors.
2. Submit a valid bundle and confirm publish success plus created identifiers.
3. Re-submit a conflicting version and confirm a clear version/immutability failure.
4. Read back the created pack from the workspace-visible list or publish response and confirm the new runnable version appears.

## Smoke Test Expectations

At minimum:

1. Backend test suites covering the new authoring/publish flow pass.
2. Existing scoring and run-loading tests still pass where touched.
3. No regression is introduced in challenge-pack listing or run creation paths.

## End-to-End Expectations

The practical end-to-end proof for this PR is:

1. Author a pack bundle in YAML.
2. Validate it locally or through the shared validator path.
3. Publish it without manual SQL/DB edits.
4. Confirm the resulting persisted pack version is runnable by the normal eval flow or is intentionally staged in a way that preserves that path for the current runtime limits.

## Review Checklist

A reviewer should be able to verify:

1. The implementation matches the locked issue comment decisions.
2. `#142` owns schema/validation/publish flow and does not silently swallow `#144` scope.
3. The code is declarative and bundle-first, not custom-code-first.
4. Errors are specific enough for authoring UX/CLI use later.
5. The final code and tests match this document with no missing promised behavior.
