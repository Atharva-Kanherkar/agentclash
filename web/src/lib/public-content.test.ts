import { describe, expect, it } from "vitest";
import {
  buildPublicLlmsFull,
  buildPublicLlmsIndex,
  canonicalPathFromMarkdownSegments,
  getAllPublicContent,
  getPublicSearchIndex,
  normalizePublicPath,
  resolvePublicContent,
} from "./public-content";

describe("public content registry", () => {
  it("has unique canonical and Markdown paths", () => {
    const items = getAllPublicContent();
    expect(new Set(items.map((item) => item.canonicalPath)).size).toBe(items.length);
    expect(new Set(items.map((item) => item.markdownPath)).size).toBe(items.length);
  });

  it("renders every registered page as semantic Markdown", () => {
    for (const item of getAllPublicContent()) {
      const markdown = item.renderMarkdown("https://example.test");
      expect(markdown, item.canonicalPath).toMatch(/^#\s+\S/m);
      expect(markdown, item.canonicalPath).toContain("Source: https://example.test");
      expect(markdown, item.canonicalPath).not.toContain("<nav");
      expect(markdown, item.canonicalPath).not.toContain("<footer");
    }
  });

  it("covers core, structured, and generated route families", () => {
    for (const pathname of [
      "/",
      "/pricing",
      "/docs",
      "/docs/getting-started/quickstart",
      "/blog",
      "/changelog",
      "/compare/agentclash-vs-langsmith",
      "/features/agent-replay",
      "/try",
      "/try/codex",
      "/tryouts",
      "/agent-opportunity",
    ]) {
      expect(resolvePublicContent(pathname), pathname).not.toBeNull();
    }
  });

  it.each([
    "/../pricing",
    "/%2e%2e/pricing",
    "/docs//quickstart",
    "/share/token",
    "/workspaces/ws_123",
    "/api/private",
    "/pricing?token=secret",
  ])("rejects unsafe or private path %s", (pathname) => {
    expect(normalizePublicPath(pathname)).toBeNull();
    expect(resolvePublicContent(pathname)).toBeNull();
  });

  it("maps Markdown segments without accepting traversal", () => {
    expect(canonicalPathFromMarkdownSegments([])).toBe("/");
    expect(canonicalPathFromMarkdownSegments(["docs", "getting-started", "quickstart"])).toBe(
      "/docs/getting-started/quickstart",
    );
    expect(canonicalPathFromMarkdownSegments(["docs", "..", "pricing"])).toBeNull();
  });

  it("drives llms and search from exact registered URLs", () => {
    const index = buildPublicLlmsIndex("https://example.test");
    const full = buildPublicLlmsFull("https://example.test");
    const search = getPublicSearchIndex();

    expect(index).toContain("https://example.test/md/pricing");
    expect(index).toContain("https://example.test/openapi.yaml");
    expect(index).toContain("https://example.test/cli-schema.json");
    expect(index).not.toContain("<agent>");
    expect(full).toContain("# AgentClash Pricing");
    expect(full).toContain("Monthly: $49/ month");
    expect(full).toContain("Annual billing: $39/ month");
    expect(full).not.toContain("/share/");
    expect(full).not.toContain("# Publication ");
    expect(search.some((item) => item.href === "/pricing")).toBe(true);
    expect(search.some((item) => item.href.startsWith("/workspaces"))).toBe(false);
  });
});
