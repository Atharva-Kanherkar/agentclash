"use client";

import { usePathname } from "next/navigation";
import { Fragment } from "react";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { Separator } from "@/components/ui/separator";
import { MobileSidebar } from "./sidebar";
import { WorkspaceSwitcher } from "./workspace-switcher";
import { UserMenu } from "./user-menu";
import type { UserMeOrganization } from "@/lib/api/types";

interface TopBarProps {
  workspaceId: string;
  organizations: UserMeOrganization[];
  displayName?: string;
  email?: string;
  avatarUrl?: string;
  orgName?: string;
}

/** Map URL segments to human-readable labels */
const segmentLabels: Record<string, string> = {
  builds: "Builds",
  deployments: "Deployments",
  "challenge-packs": "Challenge Packs",
  runs: "Runs",
  comparisons: "Comparisons",
  "release-gates": "Release Gates",
};

export function TopBar({
  workspaceId,
  organizations,
  displayName,
  email,
  avatarUrl,
  orgName,
}: TopBarProps) {
  const pathname = usePathname();

  // Build breadcrumbs from path segments after /workspaces/{id}/
  const workspacePrefix = `/workspaces/${workspaceId}`;
  const afterWorkspace = pathname.startsWith(workspacePrefix)
    ? pathname.slice(workspacePrefix.length).replace(/^\//, "")
    : "";
  const segments = afterWorkspace ? afterWorkspace.split("/") : [];

  return (
    <header className="flex h-14 items-center gap-3 border-b border-border px-4">
      <MobileSidebar workspaceId={workspaceId} />

      <WorkspaceSwitcher
        currentWorkspaceId={workspaceId}
        organizations={organizations}
      />

      {segments.length > 0 && (
        <>
          <Separator orientation="vertical" className="h-5" />
          <Breadcrumb>
            <BreadcrumbList>
              {segments.map((seg, i) => {
                const isLast = i === segments.length - 1;
                const label = segmentLabels[seg] || seg;
                const href = `${workspacePrefix}/${segments.slice(0, i + 1).join("/")}`;

                return (
                  <Fragment key={seg}>
                    <BreadcrumbItem>
                      {isLast ? (
                        <BreadcrumbPage>{label}</BreadcrumbPage>
                      ) : (
                        <BreadcrumbLink href={href}>{label}</BreadcrumbLink>
                      )}
                    </BreadcrumbItem>
                    {!isLast && <BreadcrumbSeparator />}
                  </Fragment>
                );
              })}
            </BreadcrumbList>
          </Breadcrumb>
        </>
      )}

      <div className="ml-auto">
        <UserMenu
          displayName={displayName}
          email={email}
          avatarUrl={avatarUrl}
          orgName={orgName}
        />
      </div>
    </header>
  );
}
