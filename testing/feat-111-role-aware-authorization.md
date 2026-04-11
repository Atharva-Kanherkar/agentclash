# feat/111-role-aware-authorization — Test Contract

## Functional Behavior

Upgrade authorization from membership-presence checks to role-aware checks across the existing API surface.

### Permission Matrix

| Action | org_admin (implicit) | ws_admin | ws_member | ws_viewer |
|---|---|---|---|---|
| **Reads** (list/get builds, runs, packs, replays, scorecards, artifacts, deployments) | allowed | allowed | allowed | allowed |
| **Create agent build** | allowed | allowed | allowed | denied (403) |
| **Create agent build version** | allowed | allowed | allowed | denied (403) |
| **Update agent build version** | allowed | allowed | allowed | denied (403) |
| **Validate agent build version** | allowed | allowed | allowed | allowed |
| **Mark agent build version ready** | allowed | allowed | allowed | denied (403) |
| **Create agent deployment** | allowed | allowed | allowed | denied (403) |
| **Create run** | allowed | allowed | allowed | denied (403) |
| **Publish challenge pack** | allowed | allowed | allowed | denied (403) |
| **Upload artifact** | allowed | allowed | allowed | denied (403) |

### Key behaviors

1. `workspace_viewer` can read all workspace resources but cannot perform any write/mutate operations.
2. `workspace_member` can read and create/modify business objects (builds, versions, deployments, runs, packs, artifacts).
3. `workspace_admin` can do everything a member can, plus manage memberships and workspace settings (already enforced).
4. `org_admin` gets implicit workspace_admin-equivalent access to all workspaces in their org (existing behavior, must not regress).
5. The `ensureCallerCanAccessWorkspace()` helper in agent_builds.go currently does NOT honor org_admin implicit access — this is a bug that must be fixed.
6. Validate version is read-like (non-mutating), so viewers can do it.
7. All denied operations must return HTTP 403 with `{"code":"forbidden","message":"..."}`.

## Unit Tests

### permissions_test.go

- `TestRequireWorkspaceRole_AdminAllowed` — workspace_admin passes for all actions
- `TestRequireWorkspaceRole_MemberAllowed` — workspace_member passes for member-level actions
- `TestRequireWorkspaceRole_MemberDeniedAdminActions` — workspace_member denied for admin-only actions (N/A for current scope since no infra CRUD endpoints exist yet — test the matrix entry anyway)
- `TestRequireWorkspaceRole_ViewerAllowedReads` — workspace_viewer passes for read actions
- `TestRequireWorkspaceRole_ViewerDeniedWrites` — workspace_viewer denied for all write actions
- `TestRequireWorkspaceRole_OrgAdminImplicitAccess` — org_admin (not explicit workspace member) passes for all actions
- `TestRequireWorkspaceRole_UnknownRoleDenied` — unrecognized role string is denied

### Integration into existing service tests

- `TestRunCreation_ViewerDenied` — workspace_viewer caller gets ErrForbidden from CreateRun
- `TestRunCreation_MemberAllowed` — workspace_member caller succeeds (existing test, should not regress)
- `TestAgentBuild_ViewerCanRead` — workspace_viewer can GetBuild and ListBuilds
- `TestAgentBuild_ViewerCannotCreate` — workspace_viewer gets ErrForbidden from CreateBuild
- `TestAgentBuild_ViewerCannotCreateVersion` — workspace_viewer gets 403 from createAgentBuildVersionHandler
- `TestAgentBuild_ViewerCannotMarkReady` — workspace_viewer gets 403 from markAgentBuildVersionReadyHandler
- `TestAgentBuild_OrgAdminCanAccessBuild` — org_admin without explicit workspace membership can GetBuild

## Integration / Functional Tests

N/A — no database integration tests in this change. All authorization logic is tested at the unit level with mock repos.

## Smoke Tests

N/A — authorization changes are purely additive role checks. Existing smoke tests cover the happy path.

## E2E Tests

N/A — not applicable for this change.

## Manual / cURL Tests

With the dev authenticator (AUTH_MODE=dev), verify role enforcement:

### Viewer denied on run creation
```bash
curl -s -X POST http://localhost:8080/v1/runs \
  -H "Content-Type: application/json" \
  -H "X-Agentclash-User-Id: 00000000-0000-0000-0000-000000000001" \
  -H "X-Agentclash-User-Email: viewer@test.com" \
  -H "X-Agentclash-Workspace-Memberships: WORKSPACE_ID:workspace_viewer" \
  -d '{"workspace_id":"WORKSPACE_ID","challenge_pack_version_id":"VERSION_ID","agent_deployment_ids":["DEPLOY_ID"]}'
# Expected: 403 Forbidden, body contains {"code":"forbidden",...}
```

### Member allowed on run creation
```bash
curl -s -X POST http://localhost:8080/v1/runs \
  -H "Content-Type: application/json" \
  -H "X-Agentclash-User-Id: 00000000-0000-0000-0000-000000000001" \
  -H "X-Agentclash-User-Email: member@test.com" \
  -H "X-Agentclash-Workspace-Memberships: WORKSPACE_ID:workspace_member" \
  -d '{"workspace_id":"WORKSPACE_ID","challenge_pack_version_id":"VERSION_ID","agent_deployment_ids":["DEPLOY_ID"]}'
# Expected: passes authz (may fail on validation, but NOT 403)
```

### Viewer allowed on read endpoints
```bash
curl -s http://localhost:8080/v1/workspaces/WORKSPACE_ID/runs \
  -H "X-Agentclash-User-Id: 00000000-0000-0000-0000-000000000001" \
  -H "X-Agentclash-User-Email: viewer@test.com" \
  -H "X-Agentclash-Workspace-Memberships: WORKSPACE_ID:workspace_viewer"
# Expected: 200 OK
```
