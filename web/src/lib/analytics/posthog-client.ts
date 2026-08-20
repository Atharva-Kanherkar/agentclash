/**
 * The only browser-facing PostHog adapter.
 *
 * Calls made before PostHog is configured are held in a FIFO queue. This is
 * important for App Router effects: a child pageview effect can run before the
 * provider's initialization effect. An explicitly missing key disables the
 * adapter and clears the queue so local/self-hosted builds do not retain work.
 */

import posthog from "posthog-js";
import type { WebEventName, WebEventPayloads } from "./events";
import { clearAnalyticsAttribution } from "./attribution";
import { clearAuthCompletedMarker } from "./auth-marker";

type CollectorState = "pending" | "ready" | "disabled";
type QueuedOperation = () => void;

const SAFE_UTM_KEYS = [
  "utm_source",
  "utm_medium",
  "utm_campaign",
  "utm_content",
  "utm_term",
] as const;

const SENSITIVE_PROPERTY_KEYS = new Set([
  "account_name",
  "credential",
  "credentials",
  "display_name",
  "email",
  "form_contents",
  "name",
  "password",
  "run_name",
  "secret",
]);
const EMAIL_LIKE_VALUE = /[\w.%+-]+@[\w.-]+\.[a-z]{2,}/i;

let state: CollectorState = "pending";
let queue: QueuedOperation[] = [];
let lastPageViewURL = "";
let identityGeneration = 0;

function execute(operation: QueuedOperation): void {
  try {
    operation();
  } catch (error) {
    console.warn("[analytics] PostHog operation failed", error);
  }
}

export interface InitPostHogOptions {
  apiKey: string;
  apiHost: string;
}

export interface SanitizablePostHogEvent {
  properties?: Record<string, unknown>;
}

function safeCampaignValue(value: string): string {
  const safe = value.trim().slice(0, 200);
  return EMAIL_LIKE_VALUE.test(safe) ? "" : safe;
}

function enqueue(operation: QueuedOperation): void {
  if (state === "disabled") return;
  if (state === "ready") {
    execute(operation);
    return;
  }
  queue.push(operation);
}

function drainQueue(): void {
  const pending = queue;
  queue = [];
  for (const operation of pending) execute(operation);
}

export function initPostHog({ apiKey, apiHost }: InitPostHogOptions): boolean {
  if (state === "ready") return true;
  if (typeof window === "undefined") return false;
  if (!apiKey) {
    state = "disabled";
    queue = [];
    lastPageViewURL = "";
    return false;
  }

  try {
    posthog.init(apiKey, {
      api_host: apiHost || "/ingest",
      ui_host: "https://us.posthog.com",
      capture_pageview: false,
      capture_pageleave: true,
      person_profiles: "identified_only",
      autocapture: false,
      disable_session_recording: true,
      before_send: (event) => sanitizePostHogEvent(event),
    });
  } catch (error) {
    state = "disabled";
    queue = [];
    lastPageViewURL = "";
    console.warn("[analytics] PostHog initialization failed", error);
    return false;
  }
  state = "ready";
  drainQueue();
  return true;
}

export function isPostHogReady(): boolean {
  return state === "ready";
}

export function captureWebEvent<E extends WebEventName>(
  event: E,
  properties: WebEventPayloads[E],
): void {
  enqueue(() => {
    posthog.capture(event, properties as Record<string, unknown>);
  });
}

export function capturePageView(url: string): void {
  const sanitizedURL = sanitizeAnalyticsURL(url);
  if (!sanitizedURL || sanitizedURL === lastPageViewURL) return;
  lastPageViewURL = sanitizedURL;
  enqueue(() => {
    posthog.capture("$pageview", { $current_url: sanitizedURL });
  });
}

export interface IdentifyTraits {
  org_ids?: string[];
  workspace_ids?: string[];
  is_internal?: boolean;
}

export function identifyUser(
  userId: string,
  traits: IdentifyTraits,
  setOnce: Record<string, string> = {},
): void {
  if (!userId) return;
  enqueue(() => {
    posthog.identify(userId, traits as Record<string, unknown>, setOnce);
  });
}

/** Queue an operation that must run after all earlier captures/identifies. */
export function runWhenPostHogReady(callback: () => void): void {
  enqueue(callback);
}

export function getPostHogSessionID(): string | undefined {
  if (state !== "ready") return undefined;
  return posthog.get_session_id();
}

export function getAnalyticsIdentityGeneration(): number {
  return identityGeneration;
}

/**
 * Reset aliases and generate a new anonymous device id. Local attribution and
 * session guards are cleared immediately, while the SDK reset preserves FIFO
 * order if initialization has not completed yet.
 */
export function resetPostHog(): void {
  identityGeneration += 1;
  clearAnalyticsAttribution();
  clearAuthCompletedMarker();
  lastPageViewURL = "";
  enqueue(() => posthog.reset(true));
}

export function sanitizePathname(pathname: string): string {
  const pathOnly = pathname.split(/[?#]/, 1)[0] || "/";
  const normalized = pathOnly.startsWith("/") ? pathOnly : `/${pathOnly}`;
  return normalized
    .replace(
      /\/(invites\/(?:organization|workspace)|public\/shares|agent-tryouts\/shared|share)\/[^/]+/gi,
      "/$1/{token}",
    )
    .replace(/\/auth\/callback(?:\/[^/]*)?/gi, "/auth/callback");
}

/** Keep only a safe pathname and allowlisted campaign parameters. */
export function sanitizeAnalyticsURL(raw: string): string {
  if (!raw) return "";
  const base =
    typeof window !== "undefined" ? window.location.origin : "https://analytics.invalid";
  let parsed: URL;
  try {
    parsed = new URL(raw, base);
  } catch {
    return sanitizePathname(raw.split(/[?#]/, 1)[0] || "/");
  }

  if (parsed.protocol !== "http:" && parsed.protocol !== "https:") return "";
  const safeParams = new URLSearchParams();
  for (const key of SAFE_UTM_KEYS) {
    const value = parsed.searchParams.get(key);
    const safeValue = value ? safeCampaignValue(value) : "";
    if (safeValue) safeParams.set(key, safeValue);
  }
  const query = safeParams.toString();
  const safePath = `${sanitizePathname(parsed.pathname)}${query ? `?${query}` : ""}`;
  const isAbsolute = /^[a-z][a-z\d+.-]*:\/\//i.test(raw);
  return isAbsolute ? `${parsed.origin}${safePath}` : safePath;
}

export function referrerHostname(raw: string): string {
  if (!raw) return "";
  try {
    return new URL(raw).hostname.toLowerCase();
  } catch {
    return "";
  }
}

function sanitizeProperties(
  properties: Record<string, unknown>,
): Record<string, unknown> {
  const sanitized: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(properties)) {
    const normalizedKey = key.toLowerCase();
    if (
      SENSITIVE_PROPERTY_KEYS.has(normalizedKey) ||
      normalizedKey.includes("password") ||
      normalizedKey.includes("credential") ||
      normalizedKey.includes("secret") ||
      normalizedKey.endsWith("_token")
    ) {
      continue;
    }
    if (value === undefined) continue;
    if (normalizedKey.includes("utm_")) {
      if (typeof value === "string") {
        const safeValue = safeCampaignValue(value);
        if (safeValue) sanitized[key] = safeValue;
      }
      continue;
    }
    if (normalizedKey === "$current_url" || normalizedKey.endsWith("_url")) {
      if (typeof value === "string") sanitized[key] = sanitizeAnalyticsURL(value);
      continue;
    }
    if (
      normalizedKey === "$referrer" ||
      normalizedKey === "referrer" ||
      normalizedKey.endsWith("_referrer")
    ) {
      if (typeof value === "string") {
        sanitized[key] = referrerHostname(value);
      }
      continue;
    }
    if (
      normalizedKey.endsWith("referrer_hostname") ||
      normalizedKey.endsWith("referring_domain")
    ) {
      if (typeof value === "string") {
        sanitized[key] = value.toLowerCase().slice(0, 253);
      }
      continue;
    }
    if (
      normalizedKey.endsWith("_path") ||
      normalizedKey.endsWith("pathname") ||
      normalizedKey === "path"
    ) {
      if (typeof value === "string") sanitized[key] = sanitizePathname(value);
      continue;
    }
    if (isRecord(value)) {
      sanitized[key] = sanitizeProperties(value);
      continue;
    }
    if (Array.isArray(value)) {
      sanitized[key] = value.map((item) =>
        isRecord(item) ? sanitizeProperties(item) : item,
      );
      continue;
    }
    sanitized[key] = value;
  }
  return sanitized;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export function sanitizePostHogEvent<T extends SanitizablePostHogEvent>(
  event: T | null,
): T | null {
  if (!event?.properties) return event;
  event.properties = sanitizeProperties(event.properties);
  return event;
}

/** Test-only state reset; deliberately not used by product code. */
export function resetPostHogModuleForTests(): void {
  state = "pending";
  queue = [];
  lastPageViewURL = "";
  identityGeneration = 0;
}
