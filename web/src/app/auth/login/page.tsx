import { getAuthMode } from "@/lib/auth/config";
import { DevLoginForm } from "./dev-login-form";

export const metadata = {
  title: "Sign in — AgentClash",
};

export default function LoginPage({
  searchParams,
}: {
  searchParams: Promise<{ error?: string }>;
}) {
  const authMode = getAuthMode();

  return (
    <div
      style={{
        minHeight: "100vh",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        padding: "2rem",
      }}
    >
      <div
        style={{
          width: "100%",
          maxWidth: "420px",
        }}
      >
        {/* Logo */}
        <div style={{ textAlign: "center", marginBottom: "2.5rem" }}>
          <h1
            style={{
              fontFamily: "var(--font-display), serif",
              fontSize: "2rem",
              letterSpacing: "-0.02em",
              color: "rgba(255, 255, 255, 0.9)",
              margin: 0,
            }}
          >
            AgentClash
          </h1>
          <p
            style={{
              color: "rgba(255, 255, 255, 0.4)",
              fontSize: "0.875rem",
              marginTop: "0.5rem",
            }}
          >
            Sign in to your account
          </p>
        </div>

        <ErrorBanner searchParams={searchParams} />

        {authMode === "workos" ? <WorkOSLogin /> : <DevLoginForm />}
      </div>
    </div>
  );
}

function WorkOSLogin() {
  return (
    <div
      style={{
        background: "rgba(255, 255, 255, 0.03)",
        border: "1px solid rgba(255, 255, 255, 0.08)",
        borderRadius: "12px",
        padding: "2rem",
      }}
    >
      <a
        href="/auth/login/redirect"
        style={{
          display: "block",
          width: "100%",
          padding: "0.75rem 1rem",
          background: "rgba(255, 255, 255, 0.9)",
          color: "#060606",
          borderRadius: "8px",
          fontWeight: 500,
          fontSize: "0.9375rem",
          textAlign: "center",
          textDecoration: "none",
          transition: "background 0.15s",
        }}
      >
        Sign in with WorkOS
      </a>
    </div>
  );
}

async function ErrorBanner({
  searchParams,
}: {
  searchParams: Promise<{ error?: string }>;
}) {
  const params = await searchParams;
  if (!params.error) return null;

  const messages: Record<string, string> = {
    callback_failed: "Authentication failed. Please try again.",
    session_expired: "Your session has expired. Please sign in again.",
  };

  return (
    <div
      style={{
        background: "rgba(239, 68, 68, 0.1)",
        border: "1px solid rgba(239, 68, 68, 0.2)",
        borderRadius: "8px",
        padding: "0.75rem 1rem",
        marginBottom: "1rem",
        color: "rgba(239, 68, 68, 0.9)",
        fontSize: "0.875rem",
      }}
    >
      {messages[params.error] || "An error occurred. Please try again."}
    </div>
  );
}
