import { EvalSetsClient } from "./eval-sets-client";

export default async function EvalSetsPage({
  params,
}: {
  params: Promise<{ workspaceId: string }>;
}) {
  const { workspaceId } = await params;
  return <EvalSetsClient workspaceId={workspaceId} />;
}
