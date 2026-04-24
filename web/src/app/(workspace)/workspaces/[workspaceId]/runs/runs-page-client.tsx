"use client";

import { useState } from "react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { WorkspaceWelcome } from "@/components/onboarding/workspace-welcome";
import { GlossaryTerm } from "@/components/onboarding/glossary-term";
import type { EvalSessionListItem, Run } from "@/lib/api/types";
import { RunList } from "./run-list";
import { CreateRunDialog } from "./create-run-dialog";
import { CreateEvalSessionDialog } from "./create-eval-session-dialog";
import { EvalSessionList } from "./eval-session-list";

interface RunsPageClientProps {
  workspaceId: string;
  initialRuns: Run[];
  initialTotal: number;
  initialSessions: EvalSessionListItem[];
  deploymentsCount: number;
  packsCount: number;
}

export function RunsPageClient({
  workspaceId,
  initialRuns,
  initialTotal,
  initialSessions,
  deploymentsCount,
  packsCount,
}: RunsPageClientProps) {
  const [createRunOpen, setCreateRunOpen] = useState(false);
  const openCreateRun = () => setCreateRunOpen(true);

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-lg font-semibold tracking-tight">Runs</h1>
          <p className="text-sm text-muted-foreground mt-0.5">
            Benchmark single{" "}
            <GlossaryTerm term="clash">clashes</GlossaryTerm> and repeated eval
            sessions against{" "}
            <GlossaryTerm term="challenge-pack">challenge packs</GlossaryTerm>.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <CreateEvalSessionDialog workspaceId={workspaceId} />
          <CreateRunDialog
            workspaceId={workspaceId}
            open={createRunOpen}
            onOpenChange={setCreateRunOpen}
          />
        </div>
      </div>

      <WorkspaceWelcome
        workspaceId={workspaceId}
        deploymentsCount={deploymentsCount}
        packsCount={packsCount}
        runsCount={initialTotal}
        onOpenCreateRun={openCreateRun}
      />

      <Tabs defaultValue="runs" className="w-full">
        <TabsList variant="line">
          <TabsTrigger value="runs">Runs</TabsTrigger>
          <TabsTrigger value="eval-sessions">Eval Sessions</TabsTrigger>
        </TabsList>

        <TabsContent value="runs" className="pt-4">
          <RunList
            workspaceId={workspaceId}
            initialRuns={initialRuns}
            initialTotal={initialTotal}
            onOpenCreateRun={openCreateRun}
          />
        </TabsContent>

        <TabsContent value="eval-sessions" className="pt-4">
          <EvalSessionList
            workspaceId={workspaceId}
            initialSessions={initialSessions}
          />
        </TabsContent>
      </Tabs>
    </div>
  );
}
