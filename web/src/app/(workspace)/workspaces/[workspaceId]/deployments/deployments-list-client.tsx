"use client";

import { useState } from "react";
import { Rocket } from "lucide-react";
import type { AgentDeployment } from "@/lib/api/types";
import { Badge } from "@/components/ui/badge";
import { EmptyState } from "@/components/ui/empty-state";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { CreateDeploymentDialog } from "./create-deployment-dialog";

const statusVariant: Record<string, "default" | "secondary" | "outline"> = {
  active: "default",
  paused: "outline",
  archived: "secondary",
};

interface DeploymentsListClientProps {
  workspaceId: string;
  deployments: AgentDeployment[];
}

export function DeploymentsListClient({
  workspaceId,
  deployments,
}: DeploymentsListClientProps) {
  const [dialogOpen, setDialogOpen] = useState(false);

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-lg font-semibold tracking-tight">Deployments</h1>
        <CreateDeploymentDialog
          workspaceId={workspaceId}
          open={dialogOpen}
          onOpenChange={setDialogOpen}
        />
      </div>

      {deployments.length === 0 ? (
        <EmptyState
          icon={<Rocket className="size-10" />}
          title="No deployments yet"
          description="A deployment wires a build to a model and a runtime. You'll need two to run a clash."
          action={{
            label: "Deploy your first agent",
            onClick: () => setDialogOpen(true),
          }}
        />
      ) : (
        <div className="rounded-lg border border-border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Created</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {deployments.map((d) => (
                <TableRow key={d.id}>
                  <TableCell className="font-medium">{d.name}</TableCell>
                  <TableCell>
                    <Badge variant={statusVariant[d.status] ?? "outline"}>
                      {d.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {new Date(d.created_at).toLocaleDateString()}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
