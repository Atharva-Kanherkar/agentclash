import { GitCompare } from "lucide-react";
import { ComparisonsLanding } from "./comparisons-landing";

export default async function ComparisonsPage({
  params,
}: {
  params: Promise<{ workspaceId: string }>;
}) {
  const { workspaceId } = await params;

  return (
    <div>
      <h1 className="text-lg font-semibold tracking-tight mb-4">
        Comparisons
      </h1>
      <div className="flex flex-col items-center justify-center py-12 text-center">
        <div className="mb-4 text-muted-foreground">
          <GitCompare className="size-10" />
        </div>
        <h3 className="text-sm font-medium text-foreground">
          Compare runs side-by-side
        </h3>
        <p className="mt-1 text-sm text-muted-foreground max-w-sm">
          Select two runs from the runs list to compare agent performance, or
          use the &ldquo;Compare with&hellip;&rdquo; button on any run detail
          page.
        </p>
        <ComparisonsLanding workspaceId={workspaceId} />
      </div>
    </div>
  );
}
