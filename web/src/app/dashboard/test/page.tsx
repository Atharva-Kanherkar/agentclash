"use client";

import { useAuth } from "@/lib/auth/context";

export default function ProtectedTestPage() {
  const { user } = useAuth();

  return (
    <div style={{ maxWidth: "640px" }}>
      <h1
        style={{
          fontFamily: "var(--font-display), serif",
          fontSize: "1.5rem",
          color: "rgba(255, 255, 255, 0.9)",
          letterSpacing: "-0.02em",
          marginBottom: "0.5rem",
        }}
      >
        Protected Route Test
      </h1>

      <div
        style={{
          display: "inline-flex",
          alignItems: "center",
          gap: "0.5rem",
          padding: "0.375rem 0.75rem",
          background: "rgba(34, 197, 94, 0.1)",
          border: "1px solid rgba(34, 197, 94, 0.2)",
          borderRadius: "6px",
          fontSize: "0.8125rem",
          color: "rgba(34, 197, 94, 0.9)",
          marginBottom: "2rem",
        }}
      >
        <span style={{ fontSize: "1rem" }}>&#10003;</span>
        Authenticated — you are signed in
      </div>

      <div
        style={{
          background: "rgba(255, 255, 255, 0.03)",
          border: "1px solid rgba(255, 255, 255, 0.08)",
          borderRadius: "12px",
          padding: "1.5rem",
          marginBottom: "1.5rem",
        }}
      >
        <h2
          style={{
            fontSize: "0.875rem",
            fontWeight: 500,
            color: "rgba(255, 255, 255, 0.6)",
            margin: "0 0 1rem",
            textTransform: "uppercase",
            letterSpacing: "0.05em",
          }}
        >
          Session Info
        </h2>
        <Row label="User" value={user.display_name || user.email || "Unknown"} />
        <Row label="User ID" value={user.user_id} mono />
        <Row label="Email" value={user.email || "—"} />
        <Row
          label="Orgs"
          value={user.organizations.map((o) => o.name).join(", ") || "None"}
        />
        <Row
          label="Workspaces"
          value={
            user.organizations
              .flatMap((o) => o.workspaces.map((w) => w.name))
              .join(", ") || "None"
          }
          last
        />
      </div>

      <div
        style={{
          background: "rgba(255, 255, 255, 0.03)",
          border: "1px solid rgba(255, 255, 255, 0.08)",
          borderRadius: "12px",
          padding: "1.5rem",
        }}
      >
        <h2
          style={{
            fontSize: "0.875rem",
            fontWeight: 500,
            color: "rgba(255, 255, 255, 0.6)",
            margin: "0 0 1rem",
            textTransform: "uppercase",
            letterSpacing: "0.05em",
          }}
        >
          Auth Flow Checklist
        </h2>
        <Check label="Route is protected (redirect to /auth/login if not signed in)" />
        <Check label="Session cookie is set (proxy checks existence)" />
        <Check label="Backend validated session (GET /v1/users/me succeeded)" />
        <Check label="User data loaded into AuthProvider context" />
        <Check label="useAuth() hook works in client component" />
      </div>

      <div
        style={{
          marginTop: "1.5rem",
          display: "flex",
          gap: "0.75rem",
          flexWrap: "wrap",
        }}
      >
        <NavLink href="/dashboard">Dashboard</NavLink>
        <NavLink href="/dashboard/profile">Profile</NavLink>
        <NavLink href="/">Landing (public)</NavLink>
      </div>
    </div>
  );
}

function Row({
  label,
  value,
  mono,
  last,
}: {
  label: string;
  value: string;
  mono?: boolean;
  last?: boolean;
}) {
  return (
    <div
      style={{
        display: "flex",
        justifyContent: "space-between",
        alignItems: "center",
        padding: "0.5rem 0",
        borderBottom: last ? "none" : "1px solid rgba(255, 255, 255, 0.04)",
      }}
    >
      <span style={{ fontSize: "0.8125rem", color: "rgba(255, 255, 255, 0.4)" }}>
        {label}
      </span>
      <span
        style={{
          fontSize: "0.8125rem",
          color: "rgba(255, 255, 255, 0.7)",
          fontFamily: mono ? "var(--font-mono), monospace" : "inherit",
          maxWidth: "320px",
          overflow: "hidden",
          textOverflow: "ellipsis",
        }}
      >
        {value}
      </span>
    </div>
  );
}

function Check({ label }: { label: string }) {
  return (
    <div
      style={{
        display: "flex",
        alignItems: "flex-start",
        gap: "0.5rem",
        padding: "0.375rem 0",
        fontSize: "0.8125rem",
      }}
    >
      <span style={{ color: "rgba(34, 197, 94, 0.8)", flexShrink: 0 }}>&#10003;</span>
      <span style={{ color: "rgba(255, 255, 255, 0.55)" }}>{label}</span>
    </div>
  );
}

function NavLink({ href, children }: { href: string; children: React.ReactNode }) {
  return (
    <a
      href={href}
      style={{
        padding: "0.5rem 1rem",
        background: "rgba(255, 255, 255, 0.06)",
        border: "1px solid rgba(255, 255, 255, 0.1)",
        borderRadius: "8px",
        color: "rgba(255, 255, 255, 0.6)",
        fontSize: "0.8125rem",
        textDecoration: "none",
        transition: "all 0.15s",
      }}
    >
      {children}
    </a>
  );
}
