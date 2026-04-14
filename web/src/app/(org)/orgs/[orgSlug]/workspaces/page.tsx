import { OrgWorkspacesLoader } from "./org-workspaces-loader";

export default function OrgWorkspacesPage() {
  return (
    <div>
      <h1 className="text-lg font-semibold tracking-tight mb-6">Workspaces</h1>
      <OrgWorkspacesLoader />
    </div>
  );
}
