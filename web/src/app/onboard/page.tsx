import { withAuth } from "@workos-inc/authkit-nextjs";
import { redirect } from "next/navigation";
import { getServerApiClient } from "@/lib/api/server";
import type { SessionResponse } from "@/lib/api/types";
import { OnboardingWizard } from "./onboarding-wizard";

export default async function OnboardPage() {
  const { user } = await withAuth();
  if (!user) redirect("/auth/login");

  // If user already has org memberships, they're onboarded — skip.
  try {
    const api = await getServerApiClient();
    const session = await api.get<SessionResponse>("/v1/auth/session");
    const hasOrg = session.organization_memberships.some(
      (m) => m.role === "org_admin",
    );
    if (hasOrg) {
      const firstWorkspace = session.workspace_memberships[0];
      if (firstWorkspace) {
        redirect(`/workspaces/${firstWorkspace.workspace_id}`);
      }
      redirect("/dashboard");
    }
  } catch {
    // If session fetch fails, let them proceed with onboarding —
    // the POST will return 409 if they're already onboarded.
  }

  return <OnboardingWizard />;
}
