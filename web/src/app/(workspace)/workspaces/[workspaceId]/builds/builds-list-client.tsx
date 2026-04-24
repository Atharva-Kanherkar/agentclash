"use client";

import { useState } from "react";
import Link from "next/link";
import { Bot } from "lucide-react";
import type { AgentBuild } from "@/lib/api/types";
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
import { CreateBuildDialog } from "./create-build-dialog";

const statusVariant: Record<
  string,
  "default" | "secondary" | "destructive" | "outline"
> = {
  active: "default",
  archived: "secondary",
};

interface BuildsListClientProps {
  workspaceId: string;
  builds: AgentBuild[];
}

export function BuildsListClient({
  workspaceId,
  builds,
}: BuildsListClientProps) {
  const [dialogOpen, setDialogOpen] = useState(false);

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-lg font-semibold tracking-tight">Agent Builds</h1>
        <CreateBuildDialog
          workspaceId={workspaceId}
          open={dialogOpen}
          onOpenChange={setDialogOpen}
        />
      </div>

      {builds.length === 0 ? (
        <EmptyState
          icon={<Bot className="size-10" />}
          title="No agent builds yet"
          description="A build is the spec for one agent — prompt, tools, and behavior."
          action={{
            label: "New build",
            onClick: () => setDialogOpen(true),
          }}
        />
      ) : (
        <div className="rounded-lg border border-border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Slug</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Created</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {builds.map((build) => (
                <TableRow key={build.id}>
                  <TableCell>
                    <Link
                      href={`/workspaces/${workspaceId}/builds/${build.id}`}
                      className="font-medium text-foreground hover:underline underline-offset-4"
                    >
                      {build.name}
                    </Link>
                    {build.description && (
                      <p className="text-xs text-muted-foreground mt-0.5 truncate max-w-xs">
                        {build.description}
                      </p>
                    )}
                  </TableCell>
                  <TableCell>
                    <code className="text-xs font-[family-name:var(--font-mono)] text-muted-foreground">
                      {build.slug}
                    </code>
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant={statusVariant[build.lifecycle_status] ?? "outline"}
                    >
                      {build.lifecycle_status}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {new Date(build.created_at).toLocaleDateString()}
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
