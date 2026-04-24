# Workspace Onboarding

The workspace Runs page renders a dismissible `WorkspaceWelcome` card (see
`web/src/components/onboarding/workspace-welcome.tsx`) with a 3-step checklist:

1. Deploy two agents
2. Pick a challenge pack
3. Run your first clash

Step completion is derived from the server-rendered counts already fetched on
the Runs page (`GET /v1/workspaces/:id/agent-deployments`, `/challenge-packs`,
`/runs`). The card auto-hides when all three are complete or when the user
dismisses it.

## Persistence

Dismissal is stored in localStorage and scoped per-workspace:

| Key | Meaning |
|---|---|
| `agentclash:onboarding:dismissed:<workspaceId>` | `"1"` if the user clicked the × |
| `agentclash:onboarding:first_run_seen:<workspaceId>` | `"1"` after the first-run success toast has fired |

`useOnboardingState(workspaceId)` is an SSR-safe `useSyncExternalStore` hook
that listens for both the `storage` event (cross-tab) and a
`agentclash:onboarding:sync` custom event (same-tab, cross-component).

## Development reset

To replay onboarding while developing:

- Use the **Restart onboarding** entry in the user menu (top-right avatar) — it
  calls `restartOnboarding(workspaceId)` from `use-onboarding-state.ts`.
- Or, in devtools:
  ```js
  localStorage.removeItem("agentclash:onboarding:dismissed:<workspaceId>");
  localStorage.removeItem("agentclash:onboarding:first_run_seen:<workspaceId>");
  ```

## Glossary tooltips

Plain-language definitions for terms like "deployment", "challenge pack",
"input set" live in `web/src/components/onboarding/glossary.ts` and render via
`GlossaryTerm` — a small `(?)` tooltip trigger. Used on the Runs page header
and the Create-Run dialog labels.
