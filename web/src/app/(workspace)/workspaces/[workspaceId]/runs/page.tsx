import { withAuth } from "@workos-inc/authkit-nextjs";
import { redirect } from "next/navigation";
import { createApiClient } from "@/lib/api/client";
import type {
  AgentDeployment,
  ChallengePack,
  ListEvalSessionsResponse,
  Run,
} from "@/lib/api/types";
import { RunsPageClient } from "./runs-page-client";

export default async function RunsPage({
  params,
}: {
  params: Promise<{ workspaceId: string }>;
}) {
  const { accessToken } = await withAuth();
  if (!accessToken) redirect("/auth/login");

  const { workspaceId } = await params;

  const api = createApiClient(accessToken);
  const [runsResponse, evalSessionsResponse, deploymentsResponse, packsResponse] =
    await Promise.all([
      api.get<{ items: Run[]; total: number; limit: number; offset: number }>(
        `/v1/workspaces/${workspaceId}/runs`,
        { params: { limit: 20, offset: 0 } },
      ),
      api.get<ListEvalSessionsResponse>("/v1/eval-sessions", {
        params: { workspace_id: workspaceId, limit: 20, offset: 0 },
      }),
      api.get<{ items: AgentDeployment[] }>(
        `/v1/workspaces/${workspaceId}/agent-deployments`,
      ),
      api.get<{ items: ChallengePack[] }>(
        `/v1/workspaces/${workspaceId}/challenge-packs`,
      ),
    ]);

  const activeDeploymentsCount = deploymentsResponse.items.filter(
    (d) => d.status === "active",
  ).length;
  const packsCount = packsResponse.items.length;

  return (
    <RunsPageClient
      workspaceId={workspaceId}
      initialRuns={runsResponse.items}
      initialTotal={runsResponse.total}
      initialSessions={evalSessionsResponse.items}
      deploymentsCount={activeDeploymentsCount}
      packsCount={packsCount}
    />
  );
}
