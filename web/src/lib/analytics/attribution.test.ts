import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  attributionSetOnce,
  attributionStorageKeyForTests,
  recordFirstTouch,
  rememberAcquisitionCTA,
} from "./attribution";

describe("first-touch attribution", () => {
  beforeEach(() => {
    window.localStorage.clear();
    window.history.replaceState(
      {},
      "",
      "/pricing?utm_source=google&utm_campaign=launch&utm_content=private%40example.com&email=private%40example.com",
    );
    Object.defineProperty(document, "referrer", {
      configurable: true,
      value: "https://search.example/private?q=secret",
    });
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-08-20T00:00:00Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("persists first touch and the latest acquisition CTA without PII", () => {
    recordFirstTouch();
    rememberAcquisitionCTA("pricing.closing.start_free");
    window.history.replaceState({}, "", "/enterprise?utm_source=other");
    recordFirstTouch();

    expect(attributionSetOnce()).toEqual({
      acquisition_entry_path: "/pricing",
      acquisition_referrer_hostname: "search.example",
      acquisition_utm_source: "google",
      acquisition_utm_campaign: "launch",
      acquisition_cta_id: "pricing.closing.start_free",
    });
    expect(window.localStorage.getItem(attributionStorageKeyForTests)).not.toContain(
      "private@example.com",
    );
  });

  it("expires first touch and CTA after 30 days", () => {
    recordFirstTouch();
    rememberAcquisitionCTA("pricing.closing.start_free");
    vi.advanceTimersByTime(31 * 24 * 60 * 60 * 1_000);

    expect(attributionSetOnce()).toEqual({});
  });
});
