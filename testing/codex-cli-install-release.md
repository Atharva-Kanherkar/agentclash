# codex/cli-install-release — Test Contract

## Functional Behavior
- Installation docs present truthful, working paths for macOS, Linux, Windows, and direct GitHub release downloads.
- `go install` is removed from primary user install guidance while the CLI module path still points at the old GitHub owner.
- `scripts/install/install.sh` supports Linux and macOS on amd64/arm64, respects `VERSION` and `INSTALL_DIR`, uses `curl -fsSL`, verifies release assets and `checksums.txt`, falls back to a user install directory when `/usr/local/bin` is not writable and sudo is unavailable, and prints clear uninstall guidance.
- `scripts/install/install.ps1` supports Windows amd64/arm64, respects `-Version` and `-InstallDir`, downloads the matching zip, verifies `checksums.txt`, installs to `%LOCALAPPDATA%\agentclash\bin` by default, and prints PATH guidance when needed.
- GoReleaser keeps the stable release model tag-triggered, publishes GitHub release assets for linux/darwin/windows amd64/arm64, and is configured for first-class Homebrew and Winget metadata without making prereleases overwrite stable package-manager channels.
- Release Please creates CLI-scoped release PRs from conventional commits affecting CLI, installer, or release config paths.
- A main-branch snapshot workflow builds fresh CLI artifacts on CLI-impacting merges without marking every merge as a stable release.

## Unit Tests
- N/A — this change is distribution automation, installer shell, docs, and workflow configuration.

## Integration / Functional Tests
- `cd cli && go test -short -race -count=1 ./...` passes.
- `cd cli && goreleaser check` passes with the updated GoReleaser config.
- `cd cli && goreleaser release --snapshot --clean` builds local artifacts without publishing.
- `sh -n scripts/install/install.sh` passes.
- PowerShell parses `scripts/install/install.ps1` successfully when `pwsh` is available.
- Workflow YAML parses as valid YAML.

## Smoke Tests
- The Unix installer can be run with `VERSION=<tag>` and an isolated `INSTALL_DIR` against a published release.
- The Windows installer documents the equivalent invocation and can be syntax-checked locally.
- Installed binaries should answer `agentclash version` after install.

## E2E Tests
- N/A for local automation — full package-manager publishing requires repository secrets, a Homebrew tap, and Winget review/acceptance.

## Manual / cURL Tests
- Confirm the next real release creates Linux, macOS, and Windows archives plus `checksums.txt`.
- Confirm the Homebrew tap repository `agentclash/homebrew-tap` exists and `HOMEBREW_TAP_TOKEN` can push to it before advertising Homebrew as live.
- Confirm Winget publishing credentials/tokens are available before enabling public Winget submission.
- Install matrix after release:
  - macOS Intel and Apple Silicon: Homebrew and `install.sh`.
  - Linux amd64 and arm64: Homebrew on Linux and `install.sh`.
  - Windows amd64 and arm64: `install.ps1`, direct zip, then Winget once manifest publishing is accepted.
