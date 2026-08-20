export const AUTH_COMPLETED_COOKIE = "ac_auth_completed";

export function clearAuthCompletedMarker(): void {
  if (typeof document === "undefined") return;
  document.cookie = `${AUTH_COMPLETED_COOKIE}=; Max-Age=0; Path=/; SameSite=Lax`;
}

export function consumeAuthCompletedMarker(): boolean {
  if (typeof document === "undefined") return false;
  const found = document.cookie
    .split(";")
    .map((part) => part.trim())
    .some((part) => part === `${AUTH_COMPLETED_COOKIE}=1`);
  if (found) {
    clearAuthCompletedMarker();
  }
  return found;
}
