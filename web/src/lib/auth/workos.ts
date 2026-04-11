import { WorkOS } from "@workos-inc/node";
import { getWorkOSConfig } from "./config";

let workosInstance: WorkOS | null = null;

/**
 * Returns a lazily-initialized WorkOS client singleton.
 * Only call this from server-side code (Route Handlers, Server Actions,
 * Server Components). Will throw if WorkOS env vars are missing.
 */
export function getWorkOSClient(): WorkOS {
  if (!workosInstance) {
    const cfg = getWorkOSConfig();
    workosInstance = new WorkOS(cfg.apiKey, { clientId: cfg.clientId });
  }
  return workosInstance;
}
