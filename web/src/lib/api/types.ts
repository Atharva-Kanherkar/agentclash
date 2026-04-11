/**
 * Backend API response types.
 * These mirror the Go structs defined in backend/internal/api/.
 */

// --- Auth & Session ---

/** GET /v1/auth/session — mirrors sessionResponse in routes.go */
export interface SessionResponse {
  user_id: string;
  workos_user_id?: string;
  email?: string;
  display_name?: string;
  organization_memberships: OrganizationMembership[];
  workspace_memberships: WorkspaceMembership[];
}

export interface OrganizationMembership {
  organization_id: string;
  role: string; // "org_admin" | "org_member"
}

export interface WorkspaceMembership {
  workspace_id: string;
  role: string; // "workspace_admin" | "workspace_member" | "workspace_viewer"
}

// --- Users ---

/** GET /v1/users/me — mirrors GetUserMeResult in users.go */
export interface UserMeResponse {
  user_id: string;
  workos_user_id?: string;
  email?: string;
  display_name?: string;
  organizations: UserMeOrganization[];
}

export interface UserMeOrganization {
  id: string;
  name: string;
  slug: string;
  role: string;
  workspaces: UserMeWorkspace[];
}

export interface UserMeWorkspace {
  id: string;
  name: string;
  slug: string;
  role: string;
}

// --- Onboarding ---

/** POST /v1/onboarding — mirrors OnboardResult in onboarding.go */
export interface OnboardResult {
  organization: OrganizationResult;
  workspace: WorkspaceResult;
}

export interface OrganizationResult {
  id: string;
  name: string;
  slug: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface WorkspaceResult {
  id: string;
  organization_id: string;
  name: string;
  slug: string;
  status: string;
  created_at: string;
  updated_at: string;
}

// --- Errors ---

/** Standard error envelope returned by all backend error responses. */
export interface ApiErrorResponse {
  error: {
    code: string;
    message: string;
  };
}
