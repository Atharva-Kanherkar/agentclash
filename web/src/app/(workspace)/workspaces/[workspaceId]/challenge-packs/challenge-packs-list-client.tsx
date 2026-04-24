"use client";

import { useState } from "react";
import Link from "next/link";
import { Package } from "lucide-react";
import type { ChallengePack } from "@/lib/api/types";
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
import { PublishPackDialog } from "./publish-pack-dialog";

const lifecycleVariant: Record<string, "default" | "secondary" | "outline"> = {
  runnable: "default",
  draft: "outline",
  deprecated: "secondary",
  archived: "secondary",
};

interface ChallengePacksListClientProps {
  workspaceId: string;
  packs: ChallengePack[];
}

export function ChallengePacksListClient({
  workspaceId,
  packs,
}: ChallengePacksListClientProps) {
  const [dialogOpen, setDialogOpen] = useState(false);

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-lg font-semibold tracking-tight">
            Challenge Packs
          </h1>
          <p className="text-sm text-muted-foreground mt-0.5">
            Benchmark definitions that agents are tested against.
          </p>
        </div>
        <PublishPackDialog
          workspaceId={workspaceId}
          open={dialogOpen}
          onOpenChange={setDialogOpen}
        />
      </div>

      {packs.length === 0 ? (
        <EmptyState
          icon={<Package className="size-10" />}
          title="No challenge packs"
          description="A pack is the task your agents will attempt — inputs, expected outputs, and scoring."
          action={{
            label: "Publish a pack",
            onClick: () => setDialogOpen(true),
          }}
        />
      ) : (
        <div className="rounded-lg border border-border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Description</TableHead>
                <TableHead>Versions</TableHead>
                <TableHead>Latest Status</TableHead>
                <TableHead>Created</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {packs.map((pack) => {
                const latestVersion =
                  pack.versions.length > 0
                    ? pack.versions.reduce((a, b) =>
                        a.version_number > b.version_number ? a : b,
                      )
                    : null;

                return (
                  <TableRow key={pack.id}>
                    <TableCell>
                      <Link
                        href={`/workspaces/${workspaceId}/challenge-packs/${pack.id}`}
                        className="font-medium text-foreground hover:underline underline-offset-4"
                      >
                        {pack.name}
                      </Link>
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm max-w-xs truncate">
                      {pack.description ?? "—"}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {pack.versions.length}
                    </TableCell>
                    <TableCell>
                      {latestVersion ? (
                        <Badge
                          variant={
                            lifecycleVariant[latestVersion.lifecycle_status] ??
                            "outline"
                          }
                        >
                          {latestVersion.lifecycle_status}
                        </Badge>
                      ) : (
                        <span className="text-muted-foreground text-sm">
                          {"—"}
                        </span>
                      )}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {new Date(pack.created_at).toLocaleDateString()}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
