"use client";

import { useActionState } from "react";
import { devSignIn } from "./actions";

const DEV_DEFAULTS = {
  userId: "00000000-0000-0000-0000-000000000001",
  email: "dev@agentclash.local",
  displayName: "Dev User",
  orgMemberships: "",
  workspaceMemberships: "",
};

export function DevLoginForm() {
  const [error, formAction, isPending] = useActionState(
    async (_prev: string | null, formData: FormData) => {
      try {
        await devSignIn(formData);
        return null;
      } catch (e) {
        return e instanceof Error ? e.message : "Sign-in failed";
      }
    },
    null,
  );

  return (
    <div
      style={{
        background: "rgba(255, 255, 255, 0.03)",
        border: "1px solid rgba(255, 255, 255, 0.08)",
        borderRadius: "12px",
        padding: "1.5rem",
      }}
    >
      {/* Dev mode indicator */}
      <div
        style={{
          display: "inline-flex",
          alignItems: "center",
          gap: "0.375rem",
          padding: "0.25rem 0.625rem",
          background: "rgba(250, 204, 21, 0.1)",
          border: "1px solid rgba(250, 204, 21, 0.2)",
          borderRadius: "6px",
          fontSize: "0.75rem",
          color: "rgba(250, 204, 21, 0.9)",
          marginBottom: "1.25rem",
          fontFamily: "var(--font-mono), monospace",
        }}
      >
        DEV MODE
      </div>

      <form action={formAction}>
        <Field
          label="User ID"
          name="userId"
          defaultValue={DEV_DEFAULTS.userId}
          placeholder="UUID"
          required
          mono
        />
        <Field
          label="Email"
          name="email"
          type="email"
          defaultValue={DEV_DEFAULTS.email}
          required
        />
        <Field
          label="Display Name"
          name="displayName"
          defaultValue={DEV_DEFAULTS.displayName}
        />
        <Field
          label="Org Memberships"
          name="orgMemberships"
          defaultValue={DEV_DEFAULTS.orgMemberships}
          placeholder="uuid:role,uuid:role"
          mono
        />
        <Field
          label="Workspace Memberships"
          name="workspaceMemberships"
          defaultValue={DEV_DEFAULTS.workspaceMemberships}
          placeholder="uuid:role,uuid:role"
          mono
        />

        {error && (
          <div
            style={{
              color: "rgba(239, 68, 68, 0.9)",
              fontSize: "0.8125rem",
              marginBottom: "0.75rem",
            }}
          >
            {error}
          </div>
        )}

        <button
          type="submit"
          disabled={isPending}
          style={{
            width: "100%",
            padding: "0.625rem 1rem",
            background: isPending
              ? "rgba(255, 255, 255, 0.06)"
              : "rgba(255, 255, 255, 0.9)",
            color: isPending ? "rgba(255, 255, 255, 0.4)" : "#060606",
            border: "none",
            borderRadius: "8px",
            fontWeight: 500,
            fontSize: "0.875rem",
            cursor: isPending ? "not-allowed" : "pointer",
            transition: "background 0.15s",
            marginTop: "0.25rem",
          }}
        >
          {isPending ? "Signing in..." : "Sign in as Dev User"}
        </button>
      </form>
    </div>
  );
}

function Field({
  label,
  name,
  type = "text",
  defaultValue,
  placeholder,
  required,
  mono,
}: {
  label: string;
  name: string;
  type?: string;
  defaultValue?: string;
  placeholder?: string;
  required?: boolean;
  mono?: boolean;
}) {
  return (
    <div style={{ marginBottom: "0.875rem" }}>
      <label
        htmlFor={name}
        style={{
          display: "block",
          fontSize: "0.8125rem",
          color: "rgba(255, 255, 255, 0.5)",
          marginBottom: "0.375rem",
        }}
      >
        {label}
        {required && (
          <span style={{ color: "rgba(239, 68, 68, 0.7)" }}> *</span>
        )}
      </label>
      <input
        id={name}
        name={name}
        type={type}
        defaultValue={defaultValue}
        placeholder={placeholder}
        required={required}
        style={{
          width: "100%",
          padding: "0.5rem 0.75rem",
          background: "rgba(255, 255, 255, 0.04)",
          border: "1px solid rgba(255, 255, 255, 0.1)",
          borderRadius: "6px",
          color: "rgba(255, 255, 255, 0.85)",
          fontSize: "0.8125rem",
          fontFamily: mono
            ? "var(--font-mono), monospace"
            : "var(--font-body), sans-serif",
          outline: "none",
          transition: "border-color 0.15s",
        }}
        onFocus={(e) => {
          e.currentTarget.style.borderColor = "rgba(255, 255, 255, 0.25)";
        }}
        onBlur={(e) => {
          e.currentTarget.style.borderColor = "rgba(255, 255, 255, 0.1)";
        }}
      />
    </div>
  );
}
