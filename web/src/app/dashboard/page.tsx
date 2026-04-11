"use client";

import { useAuth } from "@/lib/auth/context";

export default function DashboardPage() {
  const { user } = useAuth();

  const hasOrgs = user.organizations.length > 0;

  return (
    <div style={{ maxWidth: "720px" }}>
      <h1
        style={{
          fontFamily: "var(--font-display), serif",
          fontSize: "1.5rem",
          color: "rgba(255, 255, 255, 0.9)",
          letterSpacing: "-0.02em",
          marginBottom: "0.5rem",
        }}
      >
        Dashboard
      </h1>
      <p
        style={{
          color: "rgba(255, 255, 255, 0.4)",
          fontSize: "0.875rem",
          marginBottom: "2rem",
        }}
      >
        Welcome back, {user.display_name || user.email || "User"}
      </p>

      {!hasOrgs ? (
        <NoOrganizations />
      ) : (
        <OrganizationList organizations={user.organizations} />
      )}
    </div>
  );
}

function NoOrganizations() {
  return (
    <div
      style={{
        background: "rgba(255, 255, 255, 0.03)",
        border: "1px solid rgba(255, 255, 255, 0.08)",
        borderRadius: "12px",
        padding: "2rem",
        textAlign: "center",
      }}
    >
      <p style={{ color: "rgba(255, 255, 255, 0.5)", marginBottom: "0.5rem" }}>
        You don&apos;t belong to any organizations yet.
      </p>
      <p style={{ color: "rgba(255, 255, 255, 0.35)", fontSize: "0.875rem" }}>
        Create an organization to get started, or ask your team to invite you.
      </p>
    </div>
  );
}

function OrganizationList({
  organizations,
}: {
  organizations: {
    id: string;
    name: string;
    slug: string;
    role: string;
    workspaces: { id: string; name: string; slug: string; role: string }[];
  }[];
}) {
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
      {organizations.map((org) => (
        <div
          key={org.id}
          style={{
            background: "rgba(255, 255, 255, 0.03)",
            border: "1px solid rgba(255, 255, 255, 0.08)",
            borderRadius: "12px",
            padding: "1.25rem",
          }}
        >
          <div
            style={{
              display: "flex",
              alignItems: "center",
              justifyContent: "space-between",
              marginBottom: org.workspaces.length > 0 ? "1rem" : 0,
            }}
          >
            <div>
              <h2
                style={{
                  fontSize: "1rem",
                  fontWeight: 500,
                  color: "rgba(255, 255, 255, 0.85)",
                  margin: 0,
                }}
              >
                {org.name}
              </h2>
              <span
                style={{
                  fontFamily: "var(--font-mono), monospace",
                  fontSize: "0.75rem",
                  color: "rgba(255, 255, 255, 0.35)",
                }}
              >
                {org.slug}
              </span>
            </div>
            <RoleBadge role={org.role} />
          </div>

          {org.workspaces.length > 0 && (
            <div style={{ display: "flex", flexDirection: "column", gap: "0.5rem" }}>
              {org.workspaces.map((ws) => (
                <div
                  key={ws.id}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "space-between",
                    padding: "0.625rem 0.75rem",
                    background: "rgba(255, 255, 255, 0.03)",
                    border: "1px solid rgba(255, 255, 255, 0.06)",
                    borderRadius: "8px",
                  }}
                >
                  <div>
                    <span
                      style={{
                        fontSize: "0.875rem",
                        color: "rgba(255, 255, 255, 0.7)",
                      }}
                    >
                      {ws.name}
                    </span>
                    <span
                      style={{
                        fontFamily: "var(--font-mono), monospace",
                        fontSize: "0.6875rem",
                        color: "rgba(255, 255, 255, 0.3)",
                        marginLeft: "0.5rem",
                      }}
                    >
                      {ws.slug}
                    </span>
                  </div>
                  <RoleBadge role={ws.role} />
                </div>
              ))}
            </div>
          )}

          {org.workspaces.length === 0 && (
            <p
              style={{
                color: "rgba(255, 255, 255, 0.35)",
                fontSize: "0.8125rem",
                margin: "0.75rem 0 0",
              }}
            >
              No workspaces yet
            </p>
          )}
        </div>
      ))}
    </div>
  );
}

function RoleBadge({ role }: { role: string }) {
  const isAdmin = role.includes("admin");
  return (
    <span
      style={{
        padding: "0.1875rem 0.5rem",
        background: isAdmin
          ? "rgba(147, 130, 255, 0.1)"
          : "rgba(255, 255, 255, 0.06)",
        border: `1px solid ${
          isAdmin ? "rgba(147, 130, 255, 0.2)" : "rgba(255, 255, 255, 0.08)"
        }`,
        borderRadius: "4px",
        fontSize: "0.6875rem",
        fontFamily: "var(--font-mono), monospace",
        color: isAdmin
          ? "rgba(147, 130, 255, 0.9)"
          : "rgba(255, 255, 255, 0.45)",
      }}
    >
      {role}
    </span>
  );
}
