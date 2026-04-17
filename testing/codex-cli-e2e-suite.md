# codex/cli-e2e-suite - Test Contract

## Functional Behavior
- Provide a large installed-CLI smoke suite that can be run against a live AgentClash API after browser login.
- Default mode must be read-only: verify version, config, auth identity, token list, org/model/workspace-scoped list commands where possible, JSON output validity, and visible failures.
- Resource mode must be explicit via `--create-resources`; it may create isolated resources with a unique `codex-e2e-*` prefix.
- Resource mode must use a temporary `XDG_CONFIG_HOME` seeded with the user's current credentials so `workspace use` and config writes cannot mutate the user's real config.
- Resource mode must create and exercise an organization, workspace, workspace membership list, infrastructure resources, secrets, artifacts, challenge pack validation/publish, build/version lifecycle, playground/test-case lifecycle, and list-only deployment/run surfaces.
- Cleanup must remove or archive resources where the CLI exposes cleanup commands: delete playground test cases, delete playgrounds, delete secrets, archive runtime profiles, delete provider accounts, delete model aliases, and archive created workspace/org.
- The suite must be robust to optional or unavailable live features: it should record skips for commands that require missing prerequisites rather than crashing the shell.

## Unit Tests
N/A - this change is a shell smoke/E2E suite, not application code.

## Integration / Functional Tests
- `bash -n testing/cli-auth-smoke.sh` passes.
- `bash -n testing/cli-e2e-suite.sh` passes.
- `shellcheck testing/cli-auth-smoke.sh testing/cli-e2e-suite.sh` passes when `shellcheck` is installed.
- Read-only mode against `https://api.agentclash.dev` passes with valid logged-in credentials.
- Resource mode against `https://api.agentclash.dev` creates isolated resources, validates command outputs, and exits with a clear pass/fail summary.

## Smoke Tests
- `AGENTCLASH_API_URL=https://api.agentclash.dev ./testing/cli-auth-smoke.sh`
- `AGENTCLASH_API_URL=https://api.agentclash.dev ./testing/cli-e2e-suite.sh`
- `AGENTCLASH_API_URL=https://api.agentclash.dev ./testing/cli-e2e-suite.sh --create-resources`

## E2E Tests
- Browser login is assumed complete before running this suite.
- The suite validates the post-login CLI experience using the installed binary, live API, and temporary isolated config.

## Manual / cURL Tests
- Confirm the suite reports the installed binary path and version.
- Confirm resource mode prints the unique run prefix.
- Confirm created resources use the `codex-e2e-*` prefix.
- Confirm the user's real `~/.config/agentclash/config.yaml` is not changed by resource mode.
