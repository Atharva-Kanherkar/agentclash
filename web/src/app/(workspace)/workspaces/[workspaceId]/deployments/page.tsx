import { withAuth } from "@workos-inc/authkit-nextjs";
import { redirect } from "next/navigation";
import { createApiClient } from "@/lib/api/client";
import type { AgentDeployment } from "@/lib/api/types";
import { DeploymentsListClient } from "./deployments-list-client";

export default async function DeploymentsPage({
  params,
}: {
  params: Promise<{ workspaceId: string }>;
}) {
  const { accessToken } = await withAuth();
  if (!accessToken) redirect("/auth/login");

  const { workspaceId } = await params;

  const api = createApiClient(accessToken);
  const { items: deployments } = await api.get<{ items: AgentDeployment[] }>(
    `/v1/workspaces/${workspaceId}/agent-deployments`,
  );

  return (
    <DeploymentsListClient
      workspaceId={workspaceId}
      deployments={deployments}
    />
  );
}
