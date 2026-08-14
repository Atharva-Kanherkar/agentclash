import { describe, expect, it } from "vitest";
import {
  agentRequestLog,
  classifyAgentUserAgent,
  isMarkdownNegotiablePath,
  parseRepresentationPreference,
  prefersMarkdown,
  representationLinkHeader,
  shouldNegotiateMarkdown,
} from "./public-http";

describe("public representation negotiation", () => {
  it.each([
    [null, false],
    ["*/*", false],
    ["text/*", false],
    ["text/html", false],
    ["text/markdown", true],
    ["text/markdown;q=0", false],
    ["text/html, text/markdown", false],
    ["text/html;q=0.4, text/markdown;q=0.8", true],
    ["text/html;q=0.8, text/markdown;q=0.4", false],
    ["text/markdown;q=bogus", false],
    ["application/json, text/markdown;q=0.6", true],
  ])("handles Accept %s", (accept, expected) => {
    expect(prefersMarkdown(accept)).toBe(expected);
  });

  it("keeps the highest exact quality for duplicate ranges", () => {
    expect(
      parseRepresentationPreference(
        "text/markdown;q=0.2, text/markdown;q=0.9, text/html;q=0.5",
      ),
    ).toEqual({ markdown: 0.9, html: 0.5 });
  });

  it("negotiates public GET and HEAD only", () => {
    const base = {
      enabled: true,
      pathname: "/pricing",
      accept: "text/markdown",
    };
    expect(shouldNegotiateMarkdown({ ...base, method: "GET" })).toBe(true);
    expect(shouldNegotiateMarkdown({ ...base, method: "HEAD" })).toBe(true);
    expect(shouldNegotiateMarkdown({ ...base, method: "POST" })).toBe(false);
    expect(shouldNegotiateMarkdown({ ...base, enabled: false, method: "GET" })).toBe(false);
  });

  it.each([
    "/workspaces/ws_123",
    "/orgs/example",
    "/auth/login",
    "/api/indexnow",
    "/ingest/capture",
    "/share/capability-token",
    "/md/pricing",
    "/docs-md/getting-started/quickstart",
    "/openapi.yaml",
  ])("excludes %s", (pathname) => {
    expect(isMarkdownNegotiablePath(pathname)).toBe(false);
  });

  it.each([
    "/",
    "/pricing",
    "/docs/getting-started/quickstart",
    "/blog/example",
    "/compare/agentclash-vs-langsmith",
    "/publications/pub_123",
  ])("allows %s", (pathname) => {
    expect(isMarkdownNegotiablePath(pathname)).toBe(true);
  });

  it("advertises canonical and Markdown URLs", () => {
    expect(
      representationLinkHeader("/pricing", "https://www.agentclash.dev"),
    ).toBe(
      '<https://www.agentclash.dev/pricing>; rel="canonical", <https://www.agentclash.dev/md/pricing>; rel="alternate"; type="text/markdown"',
    );
  });
});

describe("agent request logging", () => {
  it("classifies named AI crawler families", () => {
    expect(classifyAgentUserAgent("Mozilla/5.0 compatible; GPTBot/1.2")).toBe(
      "GPTBot",
    );
    expect(classifyAgentUserAgent("Claude-SearchBot/1.0")).toBe(
      "Claude-SearchBot",
    );
    expect(classifyAgentUserAgent("Mozilla/5.0")).toBeNull();
  });

  it("builds an allowlisted payload without URL or header secrets", () => {
    const log = agentRequestLog({
      pathname: "/docs",
      method: "GET",
      accept: "text/markdown",
      userAgent: "GPTBot",
      requestId: "iad1::abc",
      servedRepresentation: "markdown",
    });
    expect(log).toEqual({
      level: "info",
      event: "agent_readable_request",
      path: "/docs",
      method: "GET",
      agent_family: "GPTBot",
      accept_class: "markdown",
      served_representation: "markdown",
      request_id: "iad1::abc",
    });
    expect(JSON.stringify(log)).not.toContain("authorization");
    expect(JSON.stringify(log)).not.toContain("cookie");
    expect(JSON.stringify(log)).not.toContain("?");
  });
});
