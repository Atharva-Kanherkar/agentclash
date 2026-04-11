export type AuthMode = "workos" | "dev";

export function getAuthMode(): AuthMode {
  const mode = process.env.NEXT_PUBLIC_AUTH_MODE;
  if (mode === "workos") return "workos";
  return "dev";
}

export function isDevMode(): boolean {
  return getAuthMode() === "dev";
}

export function getWorkOSConfig() {
  return {
    clientId: requiredEnv("WORKOS_CLIENT_ID"),
    apiKey: requiredEnv("WORKOS_API_KEY"),
    redirectUri: requiredEnv("WORKOS_REDIRECT_URI"),
  };
}

export function getApiUrl(): string {
  return process.env.API_URL || process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
}

export function getSessionSecret(): string {
  return requiredEnv("SESSION_SECRET");
}

function requiredEnv(name: string): string {
  const value = process.env[name];
  if (!value) {
    throw new Error(`Missing required environment variable: ${name}`);
  }
  return value;
}
