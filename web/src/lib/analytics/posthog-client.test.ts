import { beforeEach, describe, expect, it, vi } from "vitest";

const calls = vi.hoisted(() => [] as string[]);
const posthog = vi.hoisted(() => ({
  init: vi.fn(() => calls.push("init")),
  capture: vi.fn((event: string) => calls.push(`capture:${event}`)),
  identify: vi.fn(() => calls.push("identify")),
  reset: vi.fn(() => calls.push("reset")),
  get_session_id: vi.fn(() => "session-1"),
}));

vi.mock("posthog-js", () => ({ default: posthog }));

import {
  capturePageView,
  captureWebEvent,
  identifyUser,
  initPostHog,
  resetPostHog,
  resetPostHogModuleForTests,
  runWhenPostHogReady,
  sanitizeAnalyticsURL,
  sanitizePostHogEvent,
} from "./posthog-client";
import { WEB_EVENTS } from "./events";
import { attributionStorageKeyForTests } from "./attribution";

describe("PostHog browser adapter", () => {
  beforeEach(() => {
    resetPostHogModuleForTests();
    calls.length = 0;
    vi.clearAllMocks();
    window.localStorage.clear();
    window.sessionStorage.clear();
    window.history.replaceState({}, "", "/");
  });

  it("drains pre-init operations in FIFO order", () => {
    capturePageView("https://agentclash.dev/?utm_source=test&token=secret");
    captureWebEvent(WEB_EVENTS.TRYOUT_SESSION_ENDED, { tryout_id: "try-1" });
    identifyUser("user-1", { org_ids: ["org-1"] }, { acquisition_entry_path: "/" });
    runWhenPostHogReady(() => calls.push("callback"));
    resetPostHog();

    initPostHog({ apiKey: "phc_test", apiHost: "/ingest" });

    expect(calls).toEqual([
      "init",
      "capture:$pageview",
      "capture:web.tryout.session_ended",
      "identify",
      "callback",
      "reset",
    ]);
    expect(posthog.identify).toHaveBeenCalledWith(
      "user-1",
      { org_ids: ["org-1"] },
      { acquisition_entry_path: "/" },
    );
  });

  it("explicitly disables and clears queued work when the key is missing", () => {
    captureWebEvent(WEB_EVENTS.TRYOUT_SESSION_ENDED, { tryout_id: "discard" });

    expect(initPostHog({ apiKey: "", apiHost: "/ingest" })).toBe(false);
    captureWebEvent(WEB_EVENTS.TRYOUT_SESSION_ENDED, { tryout_id: "also-discard" });

    expect(posthog.capture).not.toHaveBeenCalled();
  });

  it("sanitizes and deduplicates identical pageviews", () => {
    initPostHog({ apiKey: "phc_test", apiHost: "/ingest" });
    const raw =
      "https://agentclash.dev/invites/workspace/private-token?utm_campaign=launch&code=oauth#secret";

    capturePageView(raw);
    capturePageView(raw);

    expect(posthog.capture).toHaveBeenCalledTimes(1);
    expect(posthog.capture).toHaveBeenCalledWith("$pageview", {
      $current_url:
        "https://agentclash.dev/invites/workspace/{token}?utm_campaign=launch",
    });
  });

  it("strips PII, tokens, unsafe queries, hashes, and full referrers", () => {
    expect(
      sanitizeAnalyticsURL(
        "https://agentclash.dev/public/shares/abc?utm_source=docs&utm_content=a%40b.com&email=a%40b.com#x",
      ),
    ).toBe("https://agentclash.dev/public/shares/{token}?utm_source=docs");

    const event = sanitizePostHogEvent({
      properties: {
        email: "person@example.com",
        display_name: "Person",
        run_name: "secret run",
        invite_token: "secret",
        $current_url: "https://agentclash.dev/auth/callback?code=secret",
        $referrer: "https://search.example/path?q=private",
        $initial_referrer: "https://first-touch.example/private?token=secret",
        $pathname: "/share/private-token?ignored=yes",
        $utm_content: "person@example.com",
        safe: "ok",
        nested: { email: "nested@example.com", status: "ready" },
        $set_once: {
          email: "person@example.com",
          acquisition_entry_path: "/invites/organization/private",
        },
      },
    });

    expect(event?.properties).toEqual({
      $current_url: "https://agentclash.dev/auth/callback",
      $referrer: "search.example",
      $initial_referrer: "first-touch.example",
      $pathname: "/share/{token}",
      safe: "ok",
      nested: { status: "ready" },
      $set_once: {
        acquisition_entry_path: "/invites/organization/{token}",
      },
    });
  });

  it("resets to a fresh device and clears attribution/session state", () => {
    window.localStorage.setItem(attributionStorageKeyForTests, "{}");
    window.sessionStorage.setItem("agentclash:analytics-session:user:s", "1");
    document.cookie = "ac_auth_completed=1; Path=/; SameSite=Lax";
    initPostHog({ apiKey: "phc_test", apiHost: "/ingest" });

    resetPostHog();

    expect(posthog.reset).toHaveBeenCalledWith(true);
    expect(window.localStorage.getItem(attributionStorageKeyForTests)).toBeNull();
    expect(window.sessionStorage.length).toBe(0);
    expect(document.cookie).not.toContain("ac_auth_completed=1");
  });
});
