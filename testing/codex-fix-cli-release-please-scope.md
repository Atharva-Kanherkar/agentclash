# codex/fix-cli-release-please-scope — Test Contract

## Functional Behavior
- Release Please stable release PRs are scoped to `cli/` commits only, rather than repo-root changes.
- The Release Please manifest tracks the CLI release stream under the `cli` package path while preserving the existing plain `v*` tag history.
- The Release Please changelog target remains [cli/CHANGELOG.md](/Users/ayush.parihar/.codex/worktrees/5492/agentclash/cli/CHANGELOG.md) via a package-relative `CHANGELOG.md` path.
- The Release Please workflow still runs when its own workflow file, config file, or manifest file changes.
- README, installer-only, and downstream release workflow changes no longer trigger the Release Please workflow by themselves.
- Maintainer docs describe stable CLI releases as coming from releasable `cli/` commits only.
- Existing tag-triggered publish behavior in `.github/workflows/release-cli.yml` is unchanged.
- Existing snapshot behavior in `.github/workflows/cli-snapshot.yml` is unchanged.

## Unit Tests
- N/A — workflow/config/docs change with no repo unit-test surface.

## Integration / Functional Tests
- `.github/release-please-config.json` parses as valid JSON.
- `.github/.release-please-manifest.json` parses as valid JSON.
- `git diff` shows Release Please package path moved from `"."` to `"cli"` and manifest version tracking moved to the `cli` key.
- `git diff` shows the Release Please workflow `paths` list is narrowed to `cli/**` plus the release-please workflow/config/manifest files.
- `git diff` shows maintainer-facing docs no longer claim installer/docs/release-config changes auto-open stable CLI release PRs.

## Smoke Tests
- `rg -n "CLI-impacting paths|affecting CLI, installer, or release config paths|Release Please watches CLI-impacting paths" README.md docs npm testing CLAUDE.md .github` returns no stale release-scope language.
- Manual review confirms `.github/workflows/cli-snapshot.yml` and `.github/workflows/release-cli.yml` are unchanged.

## E2E Tests
- N/A — confirming the post-merge GitHub-side reconciliation for PR `#402` is a manual maintainer follow-up outside the local repo change.

## Manual / cURL Tests
- `python3 - <<'PY'` with `json.load` for both release-please JSON files.
- `git diff -- .github/release-please-config.json .github/.release-please-manifest.json .github/workflows/release-please.yml README.md docs/cli-distribution.md npm/cli/README.md CLAUDE.md testing/codex-cli-install-release.md testing/codex-fix-release-please-tag-prefix.md testing/codex-fix-cli-release-please-scope.md`
- After merge to `main`, close PR `#402`, clear any lingering `autorelease` label, and rerun the `Release Please CLI` workflow once to reconcile the bot state.
