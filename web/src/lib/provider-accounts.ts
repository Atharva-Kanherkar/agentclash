import type { ProviderAccount } from "@/lib/api/types";

/** Returns only the network host (and optional port), never endpoint path data. */
export function providerAccountEndpointHost(
  account: Pick<ProviderAccount, "base_url">,
): string {
  const baseURL = account.base_url?.trim();
  if (!baseURL) return "";

  try {
    return new URL(baseURL).host;
  } catch {
    return "";
  }
}

/** Shared human-readable label for every provider-account selector. */
export function providerAccountLabel(
  account: Pick<ProviderAccount, "name" | "provider_key" | "base_url">,
): string {
  const endpointHost = providerAccountEndpointHost(account);
  const detail = endpointHost
    ? `${account.provider_key} · ${endpointHost}`
    : account.provider_key;
  return `${account.name} (${detail})`;
}
