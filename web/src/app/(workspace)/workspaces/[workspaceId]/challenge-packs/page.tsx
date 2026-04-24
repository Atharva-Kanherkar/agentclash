import { withAuth } from "@workos-inc/authkit-nextjs";
import { redirect } from "next/navigation";
import { createApiClient } from "@/lib/api/client";
import type { ChallengePack } from "@/lib/api/types";
import { ChallengePacksListClient } from "./challenge-packs-list-client";

export default async function ChallengePacksPage({
  params,
}: {
  params: Promise<{ workspaceId: string }>;
}) {
  const { accessToken } = await withAuth();
  if (!accessToken) redirect("/auth/login");

  const { workspaceId } = await params;

  const api = createApiClient(accessToken);
  const { items: packs } = await api.get<{ items: ChallengePack[] }>(
    `/v1/workspaces/${workspaceId}/challenge-packs`,
  );

  return <ChallengePacksListClient workspaceId={workspaceId} packs={packs} />;
}
