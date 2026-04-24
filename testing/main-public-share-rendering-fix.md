# Main Public Share Rendering Fix — Test Contract

## Functional Behavior
- Challenge pack share links render the pack version as formatted YAML, not raw JSON.
- Run scorecard share links render a public run page with the same core run UI shape: run metadata, scorecard, and all participating agents.
- Per-agent replay share links render the replay experience, including step-level replay evidence, rather than a raw JSON dump.
- Per-agent scorecard share links render the scorecard experience rather than a raw JSON dump.
- Share buttons must use the exact resource id from the page they are on and must not collapse to one stale replay or agent.
- Public pages must remain read-only, unauthenticated, and must not leak workspace or organization ids.

## Unit Tests
- Backend public share manager tests should verify distinct replay shares return distinct run-agent ids.
- Backend public payload tests should continue verifying workspace id is omitted.

## Integration / Functional Tests
- `go test ./...` from `backend/` passes.
- `pnpm lint` from `web/` passes.
- `pnpm build` from `web/` passes and includes `/share/[token]`.

## Smoke Tests
- Creating a challenge-pack share returns `/share/<token>` and the page displays YAML.
- Creating a run scorecard share returns `/share/<token>` and the page displays all participants.
- Creating a per-agent replay share returns `/share/<token>` and the page displays replay steps for that agent.
- Creating a per-agent scorecard share returns `/share/<token>` and the page displays scorecard metrics for that agent.

## Manual / cURL Tests
```bash
curl "$API_URL/public/shares/$TOKEN"
# Expected: 200, resource.type matches the shared resource, and no workspace_id/organization_id keys.
```
