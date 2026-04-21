# codex/issue-376-cli-dx — Test Contract

## Functional Behavior
- `agentclash context` shows the resolved active organization and workspace, including whether each value came from flags, environment, project config, or user config.
- `agentclash context set` accepts organization and workspace selectors as IDs, names, or slugs, persists the chosen defaults, and keeps the active org/workspace pair consistent.
- `agentclash context pick` provides an interactive TTY flow for choosing the active organization and workspace without editing raw config files.
- `agentclash context clear` can clear saved org and workspace defaults without touching environment or project config.
- `agentclash auth status` includes the currently active org/workspace context alongside authentication details.
- `agentclash org get`, `agentclash workspace get`, `agentclash workspace use`, `agentclash org members list`, and `agentclash org members invite` accept human-friendly selectors instead of forcing raw UUIDs on the common path.
- `agentclash workspace list --org ...` accepts organization ID, name, or slug, and falls back to the active org context when `--org` is omitted.
- Commands that require a workspace launch interactive recovery in a TTY when no workspace context is available; non-interactive execution still fails with a clear error.
- `agentclash init` can bind the current repo by using current context or by interactively selecting org/workspace when needed.
- `agentclash run create` can complete the happy path without manual UUID copying by guiding the user through challenge-pack version, optional input set, and deployment selection in a TTY.
- `agentclash run create` also accepts human-friendly challenge-pack and deployment selectors when passed explicitly.
- `agentclash compare runs` and `agentclash compare gate` offer a human-friendly path by resolving recent runs interactively in a TTY and by supporting non-UUID shortcuts for recent-run selection.
- README and command help text teach the context-first interactive path before the exact-ID automation path.

## Unit Tests
- Resolver tests cover organization and workspace lookup by ID, name, and slug, including ambiguity and not-found cases.
- Context resolution tests cover precedence reporting for flags, environment, project config, and user config.
- Recent-run selector tests cover shortcut resolution and ambiguity handling.
- Interactive selection helper tests cover list rendering, input validation, and cancellation/error flows.

## Integration / Functional Tests
- Command tests verify `context`, `context set`, `context clear`, and `context pick` behavior against fake API responses and isolated config directories.
- Command tests verify `workspace use`, `workspace get`, `org get`, and `org members list` resolve non-ID selectors through the shared resolver layer.
- Command tests verify missing-workspace recovery behaves differently in TTY and non-TTY execution.
- Command tests verify `init` writes `.agentclash.yaml` after resolving org/workspace through context or interactive selection.
- Command tests verify `run create` fetches challenge packs, input sets, and deployments as needed, then submits the correct resolved IDs.
- Command tests verify `compare gate` and `compare runs` resolve recent runs before calling the existing API endpoints.

## Smoke Tests
- `agentclash auth status` shows authentication details plus active context.
- `agentclash context` shows the active org/workspace after `context set` or `workspace use`.
- `agentclash init` creates `.agentclash.yaml` without passing raw IDs when context is already set.
- `agentclash run create` can be completed through guided prompts without copying UUIDs from previous command output.
- `agentclash compare gate` can be completed through guided recent-run selection without copying UUIDs manually.

## E2E Tests
- N/A — this change is covered by CLI command tests and targeted manual terminal verification.

## Manual / cURL Tests
```bash
agentclash auth status
# Expected: user details plus active org/workspace context and source information.

agentclash context set --org acme --workspace prod
agentclash context
# Expected: saved context resolves to the chosen org/workspace without raw UUID editing.

agentclash init
# Expected in a TTY with no project config: guided org/workspace selection and a created .agentclash.yaml file.

agentclash run create
# Expected in a TTY: prompts for challenge pack version, optional input set, and deployments, then creates the run.

agentclash compare gate
# Expected in a TTY: prompts for baseline/candidate recent runs, then evaluates the release gate.
```
