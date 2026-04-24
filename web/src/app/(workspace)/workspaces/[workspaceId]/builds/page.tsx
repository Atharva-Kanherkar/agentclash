import { withAuth } from "@workos-inc/authkit-nextjs";
import { redirect } from "next/navigation";
import { createApiClient } from "@/lib/api/client";
import type { AgentBuild } from "@/lib/api/types";
import { BuildsListClient } from "./builds-list-client";

export default async function BuildsPage({
  params,
}: {
  params: Promise<{ workspaceId: string }>;
}) {
  const { accessToken } = await withAuth();
  if (!accessToken) redirect("/auth/login");

  const { workspaceId } = await params;

  const api = createApiClient(accessToken);
  const { items: builds } = await api.get<{ items: AgentBuild[] }>(
    `/v1/workspaces/${workspaceId}/agent-builds`,
  );

  return <BuildsListClient workspaceId={workspaceId} builds={builds} />;
}
