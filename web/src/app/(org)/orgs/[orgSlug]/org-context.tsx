"use client";

import { createContext, useContext } from "react";

export interface OrgContextValue {
  orgId: string;
  orgSlug: string;
  orgName: string;
  isAdmin: boolean;
  currentUserId: string;
}

const OrgContext = createContext<OrgContextValue | null>(null);

export function OrgProvider({
  value,
  children,
}: {
  value: OrgContextValue;
  children: React.ReactNode;
}) {
  return <OrgContext.Provider value={value}>{children}</OrgContext.Provider>;
}

export function useOrgContext(): OrgContextValue {
  const ctx = useContext(OrgContext);
  if (!ctx) {
    throw new Error("useOrgContext must be used within an OrgProvider");
  }
  return ctx;
}
