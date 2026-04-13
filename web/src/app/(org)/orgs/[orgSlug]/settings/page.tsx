import { withAuth } from "@workos-inc/authkit-nextjs";
import { redirect } from "next/navigation";
import { OrgSettingsGate } from "./org-settings-gate";

export default async function OrgSettingsPage({
  params,
}: {
  params: Promise<{ orgSlug: string }>;
}) {
  const { accessToken } = await withAuth();
  if (!accessToken) redirect("/auth/login");

  const { orgSlug } = await params;

  return <OrgSettingsGate orgSlug={orgSlug} />;
}
