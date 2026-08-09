import { EvalSetDetailClient } from "./eval-set-detail-client";

export default async function EvalSetDetailPage({
  params,
}: {
  params: Promise<{ workspaceId: string; evalSetId: string }>;
}) {
  const { workspaceId, evalSetId } = await params;
  return (
    <EvalSetDetailClient workspaceId={workspaceId} evalSetId={evalSetId} />
  );
}
