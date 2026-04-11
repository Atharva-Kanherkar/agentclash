package api

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Action represents an authorization action in the permission matrix.
type Action string

const (
	// Read actions — allowed for all workspace roles including viewer.
	ActionReadWorkspace Action = "read_workspace"

	// Member-level write actions — allowed for workspace_admin and workspace_member.
	ActionCreateAgentBuild        Action = "create_agent_build"
	ActionCreateAgentBuildVersion Action = "create_agent_build_version"
	ActionUpdateAgentBuildVersion Action = "update_agent_build_version"
	ActionMarkAgentBuildReady     Action = "mark_agent_build_ready"
	ActionCreateAgentDeployment   Action = "create_agent_deployment"
	ActionCreateRun               Action = "create_run"
	ActionPublishChallengePack    Action = "publish_challenge_pack"
	ActionUploadArtifact          Action = "upload_artifact"

	// Admin-level actions — allowed for workspace_admin only.
	// Infrastructure CRUD endpoints don't exist yet, but the matrix
	// entry is defined so new endpoints can use it immediately.
	ActionManageInfrastructure Action = "manage_infrastructure"
)

// Workspace roles.
const (
	RoleWorkspaceAdmin  = "workspace_admin"
	RoleWorkspaceMember = "workspace_member"
	RoleWorkspaceViewer = "workspace_viewer"
	RoleOrgAdmin        = "org_admin"
	RoleOrgMember       = "org_member"
)

// permissionMatrix maps each workspace role to its allowed actions.
// org_admin is handled separately via implicit access (treated as workspace_admin).
var permissionMatrix = map[string]map[Action]bool{
	RoleWorkspaceAdmin: {
		ActionReadWorkspace:           true,
		ActionCreateAgentBuild:        true,
		ActionCreateAgentBuildVersion: true,
		ActionUpdateAgentBuildVersion: true,
		ActionMarkAgentBuildReady:     true,
		ActionCreateAgentDeployment:   true,
		ActionCreateRun:               true,
		ActionPublishChallengePack:    true,
		ActionUploadArtifact:          true,
		ActionManageInfrastructure:    true,
	},
	RoleWorkspaceMember: {
		ActionReadWorkspace:           true,
		ActionCreateAgentBuild:        true,
		ActionCreateAgentBuildVersion: true,
		ActionUpdateAgentBuildVersion: true,
		ActionMarkAgentBuildReady:     true,
		ActionCreateAgentDeployment:   true,
		ActionCreateRun:               true,
		ActionPublishChallengePack:    true,
		ActionUploadArtifact:          true,
	},
	RoleWorkspaceViewer: {
		ActionReadWorkspace: true,
	},
}

// RequireWorkspaceRole checks that the caller has sufficient role for the given
// action in the specified workspace. It checks:
//  1. Explicit workspace membership with a role that permits the action.
//  2. org_admin of the workspace's parent org (implicit workspace_admin access).
//
// The orgLookup is optional — pass nil if org_admin implicit access should not
// be checked (e.g., in tests without a database).
func RequireWorkspaceRole(
	ctx context.Context,
	caller Caller,
	workspaceID uuid.UUID,
	action Action,
	orgLookup WorkspaceOrgLookup,
) error {
	// Check 1: explicit workspace membership.
	if m, ok := caller.WorkspaceMemberships[workspaceID]; ok {
		if roleAllows(m.Role, action) {
			return nil
		}
		return fmt.Errorf("%w: role %s cannot perform %s in workspace %s", ErrForbidden, m.Role, action, workspaceID)
	}

	// Check 2: org_admin of parent org gets implicit workspace_admin access.
	if orgLookup != nil {
		orgID, err := orgLookup.GetOrganizationIDByWorkspaceID(ctx, workspaceID)
		if err == nil {
			if m, ok := caller.OrganizationMemberships[orgID]; ok && m.Role == RoleOrgAdmin {
				// org_admin is treated as workspace_admin for permission purposes.
				if roleAllows(RoleWorkspaceAdmin, action) {
					return nil
				}
			}
		}
	}

	return fmt.Errorf("%w: caller %s does not have access to workspace %s", ErrForbidden, caller.UserID, workspaceID)
}

// AuthorizeWorkspaceAction verifies workspace access via the authorizer (which
// handles org_admin implicit access) and then checks the caller's role permits
// the action. Use this in handlers that are NOT behind authorizeWorkspaceAccess
// middleware. For handlers behind middleware, the workspace access is already
// checked and this still works correctly (the AuthorizeWorkspace call is
// idempotent).
func AuthorizeWorkspaceAction(
	ctx context.Context,
	authorizer WorkspaceAuthorizer,
	caller Caller,
	workspaceID uuid.UUID,
	action Action,
) error {
	// Step 1: verify the caller can access the workspace at all.
	if err := authorizer.AuthorizeWorkspace(ctx, caller, workspaceID); err != nil {
		return err
	}

	// Step 2: check role for the specific action.
	// If they have explicit membership, check that role.
	if m, ok := caller.WorkspaceMemberships[workspaceID]; ok {
		if roleAllows(m.Role, action) {
			return nil
		}
		return fmt.Errorf("%w: role %s cannot perform %s in workspace %s", ErrForbidden, m.Role, action, workspaceID)
	}

	// No explicit membership but AuthorizeWorkspace passed → caller is org_admin.
	// org_admin is treated as workspace_admin.
	if roleAllows(RoleWorkspaceAdmin, action) {
		return nil
	}
	return fmt.Errorf("%w: insufficient privileges for %s in workspace %s", ErrForbidden, action, workspaceID)
}

// roleAllows checks whether a workspace role permits the given action.
func roleAllows(role string, action Action) bool {
	allowed, ok := permissionMatrix[role]
	if !ok {
		return false
	}
	return allowed[action]
}
