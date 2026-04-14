"use server";

import { createApiClient } from "@/lib/api/client";

interface AuthorizeResult {
  ok: boolean;
  error?: string;
}

export async function authorizeDevice(
  userCode: string,
  accessToken: string,
): Promise<AuthorizeResult> {
  try {
    const api = createApiClient(accessToken);
    await api.post("/v1/auth/device/approve", {
      user_code: userCode.toUpperCase().replace(/[^A-Z0-9]/g, "").slice(0, 4) + "-" + userCode.toUpperCase().replace(/[^A-Z0-9]/g, "").slice(4, 8),
    });
    return { ok: true };
  } catch (err) {
    return {
      ok: false,
      error: err instanceof Error ? err.message : "Failed to authorize device",
    };
  }
}
