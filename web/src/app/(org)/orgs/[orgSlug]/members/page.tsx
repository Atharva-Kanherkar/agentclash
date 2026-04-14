import { OrgMembersLoader } from "./org-members-loader";

export default function OrgMembersPage() {
  return (
    <div>
      <h1 className="text-lg font-semibold tracking-tight mb-6">Members</h1>
      <OrgMembersLoader />
    </div>
  );
}
