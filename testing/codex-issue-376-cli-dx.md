# codex/issue-376-cli-dx — Test Contract

## Functional Behavior
- `agentclash run create` can complete the happy path in a TTY without manual UUID or name copying by guiding the user through challenge-pack, challenge-pack version, optional input set, and deployment selection.
- Interactive selection uses a real scrollable terminal picker: up/down to move, Enter to confirm, and multi-select behavior for deployment selection.
- When there is only one valid next choice, the CLI auto-selects it and keeps moving instead of prompting unnecessarily.
- When multiple challenge pack versions or input sets exist, the CLI presents the picker instead of asking the user to rerun the command with copied identifiers.
- When `run create` is executed non-interactively, the existing explicit-flag path remains available and clear validation errors still explain which inputs are required.
- The README quickstart and `run create --help` teach the interactive happy path first, while preserving the exact-ID automation path for scripts and CI.

## Unit Tests
- Interactive picker tests cover single-select and multi-select flows, including empty lists and cancellation/error handling.
- Guided run-creation helper tests cover automatic selection when only one option exists and prompted selection when multiple options exist.
- Selection mapping tests cover converting displayed pack/version/deployment choices into the resolved IDs sent to the API.

## Integration / Functional Tests
- Command tests verify `run create` fetches challenge packs, input sets, and deployments as needed, then submits the correct resolved IDs after interactive selection.
- Command tests verify `run create` skips prompts when required IDs are already supplied explicitly.
- Command tests verify multi-deployment selection sends the chosen deployment IDs in the order returned by the picker.

## Smoke Tests
- `agentclash run create` can be completed through guided prompts without copying UUIDs from previous command output.
- Challenge-pack selection is navigable with arrow keys and Enter in a normal terminal session.
- Deployment selection is navigable in-terminal without having to rerun the command or paste identifiers.

## E2E Tests
- N/A — this change is covered by CLI command tests and targeted manual terminal verification.

## Manual / cURL Tests
```bash
agentclash run create
# Expected in a TTY: scrollable picker for challenge pack, version, optional input set, and deployments, then run creation with no copy/paste step.

agentclash run create \
  --challenge-pack-version cpv-123 \
  --deployments dep-a,dep-b
# Expected in non-interactive use: existing explicit-ID flow still works without prompts.
```
