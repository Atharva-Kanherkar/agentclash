"use client";

import { useAuth } from "@/lib/auth/context";

export default function ProfilePage() {
  const { user, signOut } = useAuth();

  return (
    <div style={{ maxWidth: "640px" }}>
      <h1
        style={{
          fontFamily: "var(--font-display), serif",
          fontSize: "1.5rem",
          color: "rgba(255, 255, 255, 0.9)",
          letterSpacing: "-0.02em",
          marginBottom: "1.5rem",
        }}
      >
        Profile
      </h1>

      {/* User info card */}
      <div
        style={{
          background: "rgba(255, 255, 255, 0.03)",
          border: "1px solid rgba(255, 255, 255, 0.08)",
          borderRadius: "12px",
          padding: "1.5rem",
          marginBottom: "1.5rem",
        }}
      >
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: "1rem",
            marginBottom: "1.5rem",
            paddingBottom: "1.25rem",
            borderBottom: "1px solid rgba(255, 255, 255, 0.06)",
          }}
        >
          {/* Avatar placeholder */}
          <div
            style={{
              width: "48px",
              height: "48px",
              borderRadius: "50%",
              background: "rgba(147, 130, 255, 0.15)",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              fontSize: "1.25rem",
              color: "rgba(147, 130, 255, 0.9)",
              fontWeight: 500,
              flexShrink: 0,
            }}
          >
            {(user.display_name || user.email || "U").charAt(0).toUpperCase()}
          </div>
          <div>
            <div
              style={{
                fontSize: "1.125rem",
                fontWeight: 500,
                color: "rgba(255, 255, 255, 0.9)",
              }}
            >
              {user.display_name || "No display name"}
            </div>
            <div style={{ fontSize: "0.8125rem", color: "rgba(255, 255, 255, 0.4)" }}>
              {user.email || "No email"}
            </div>
          </div>
        </div>

        <Row label="User ID" value={user.user_id} mono />
        <Row label="WorkOS User ID" value={user.workos_user_id || "—"} mono />
        <Row label="Email" value={user.email || "—"} />
        <Row label="Display Name" value={user.display_name || "—"} />
        <Row
          label="Organizations"
          value={`${user.organizations.length} org${user.organizations.length !== 1 ? "s" : ""}`}
        />
        <Row
          label="Total Workspaces"
          value={String(
            user.organizations.reduce((sum, org) => sum + org.workspaces.length, 0),
          )}
          last
        />
      </div>

      {/* Raw JSON dump for debugging */}
      <details
        style={{
          background: "rgba(255, 255, 255, 0.03)",
          border: "1px solid rgba(255, 255, 255, 0.08)",
          borderRadius: "12px",
          padding: "1rem 1.25rem",
          marginBottom: "1.5rem",
        }}
      >
        <summary
          style={{
            cursor: "pointer",
            fontSize: "0.8125rem",
            color: "rgba(255, 255, 255, 0.5)",
            userSelect: "none",
          }}
        >
          Raw API response (GET /v1/users/me)
        </summary>
        <pre
          style={{
            marginTop: "1rem",
            padding: "1rem",
            background: "rgba(0, 0, 0, 0.3)",
            borderRadius: "8px",
            fontSize: "0.75rem",
            fontFamily: "var(--font-mono), monospace",
            color: "rgba(255, 255, 255, 0.6)",
            overflow: "auto",
            maxHeight: "400px",
            lineHeight: 1.6,
          }}
        >
          {JSON.stringify(user, null, 2)}
        </pre>
      </details>

      <button
        onClick={signOut}
        style={{
          padding: "0.625rem 1.25rem",
          background: "rgba(239, 68, 68, 0.1)",
          border: "1px solid rgba(239, 68, 68, 0.2)",
          borderRadius: "8px",
          color: "rgba(239, 68, 68, 0.9)",
          fontSize: "0.875rem",
          cursor: "pointer",
          transition: "all 0.15s",
        }}
      >
        Sign out
      </button>
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
          whiteSpace: "nowrap",
        }}
      >
        {value}
      </span>
    </div>
  );
}
