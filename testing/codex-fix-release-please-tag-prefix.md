# codex/fix-release-please-tag-prefix — Test Contract

## Functional Behavior
- Release Please must use existing plain `v*` release tags, not component-prefixed tags.
- The next generated release PR must compare `v0.1.2...v0.2.0`.
- The generated release metadata must not reference `agentclash-v0.1.2` or `agentclash-v0.2.0`.
- The stale Release Please PR must not be merged while it contains component-prefixed release metadata.

## Unit Tests
- N/A — configuration-only release automation fix.

## Integration / Functional Tests
- Validate `.github/release-please-config.json` parses as JSON.
- Verify Release Please rerun logs look for `v0.1.2`, not `agentclash-v0.1.2`.
- Verify the corrected release PR keeps `.github/.release-please-manifest.json` moving `"."` from `0.1.2` to `0.2.0`.

## Smoke Tests
- Confirm no `agentclash-v0.2.0` tag or release exists before the corrected release PR is merged.
- Confirm merging the corrected Release Please PR creates `v0.2.0`.
- Confirm the `Release CLI` workflow runs from the `v0.2.0` tag.

## E2E Tests
- After release completes, install `v0.2.0` with `scripts/install/install.sh` into a temporary directory and run `agentclash version`.
- After Homebrew tap publishing completes, run `brew install --cask agentclash/tap/agentclash` and `agentclash version`.

## Manual / cURL Tests
- `gh pr view 331 --repo agentclash/agentclash --json isDraft,body`
- `gh workflow run release-please.yml --repo agentclash/agentclash --ref main`
- `gh run list --repo agentclash/agentclash --workflow "Release Please CLI" -L 5`
- `gh release view v0.2.0 --repo agentclash/agentclash`
- `git ls-remote --tags origin | grep 'refs/tags/v0.2.0'`
