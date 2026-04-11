"use client";

import { useAuth } from "@/lib/auth/context";
import type { UserMeResponse } from "@/lib/api/types";

/**
 * Minimal dashboard shell — topbar with user info and sign-out.
 * This will evolve into a full sidebar layout as features are added.
 */
export function DashboardShell({
  user,
  children,
}: {
  user: UserMeResponse;
  children: React.ReactNode;
}) {
  const { signOut } = useAuth();

  return (
    <div style={{ minHeight: "100vh", display: "flex", flexDirection: "column" }}>
      {/* Topbar */}
      <header
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          padding: "0 1.5rem",
          height: "56px",
          borderBottom: "1px solid rgba(255, 255, 255, 0.08)",
          flexShrink: 0,
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: "1.5rem" }}>
          <a
            href="/dashboard"
            style={{
              fontFamily: "var(--font-display), serif",
              fontSize: "1.125rem",
              color: "rgba(255, 255, 255, 0.9)",
              textDecoration: "none",
              letterSpacing: "-0.01em",
            }}
          >
            AgentClash
          </a>
          <nav style={{ display: "flex", gap: "0.25rem" }}>
            <NavTab href="/dashboard">Home</NavTab>
            <NavTab href="/dashboard/profile">Profile</NavTab>
            <NavTab href="/dashboard/test">Auth Test</NavTab>
          </nav>
        </div>

        <div style={{ display: "flex", alignItems: "center", gap: "1rem" }}>
          <div
            style={{
              fontSize: "0.8125rem",
              color: "rgba(255, 255, 255, 0.5)",
            }}
          >
            <span style={{ color: "rgba(255, 255, 255, 0.7)" }}>
              {user.display_name || user.email || "User"}
            </span>
            {user.email && user.display_name && (
              <span style={{ marginLeft: "0.5rem" }}>
                {user.email}
              </span>
            )}
          </div>
          <button
            onClick={signOut}
            style={{
              padding: "0.375rem 0.75rem",
              background: "rgba(255, 255, 255, 0.06)",
              border: "1px solid rgba(255, 255, 255, 0.1)",
              borderRadius: "6px",
              color: "rgba(255, 255, 255, 0.6)",
              fontSize: "0.8125rem",
              cursor: "pointer",
              transition: "all 0.15s",
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.background = "rgba(255, 255, 255, 0.1)";
              e.currentTarget.style.color = "rgba(255, 255, 255, 0.8)";
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.background = "rgba(255, 255, 255, 0.06)";
              e.currentTarget.style.color = "rgba(255, 255, 255, 0.6)";
            }}
          >
            Sign out
          </button>
        </div>
      </header>

      {/* Content */}
      <main style={{ flex: 1, padding: "2rem 1.5rem" }}>
        {children}
      </main>
    </div>
  );
}

function NavTab({ href, children }: { href: string; children: React.ReactNode }) {
  return (
    <a
      href={href}
      style={{
        padding: "0.375rem 0.75rem",
        borderRadius: "6px",
        fontSize: "0.8125rem",
        color: "rgba(255, 255, 255, 0.5)",
        textDecoration: "none",
        transition: "all 0.15s",
      }}
    >
      {children}
    </a>
  );
}
