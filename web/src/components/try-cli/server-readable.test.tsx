import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import { TryCliDemoClient } from "./demo-client";
import { TryCliLandingClient } from "./landing-client";
import type { DemoMeta } from "@/lib/try-cli/types";

vi.mock("next/link", () => ({
  default: ({ href, children, ...props }: React.AnchorHTMLAttributes<HTMLAnchorElement>) => (
    <a href={String(href)} {...props}>{children}</a>
  ),
}));

const demo: DemoMeta = {
  slug: "codex",
  name: "Codex CLI",
  tagline: "OpenAI coding agent",
  category: "AI coding agents",
  commands: [
    { label: "Show version", run: "codex --version" },
    { label: "Start Codex", run: "codex" },
  ],
  sessionMinutes: 10,
};

describe("server-readable Try CLI", () => {
  it("renders the supplied demo catalog before hydration", () => {
    const html = renderToStaticMarkup(<TryCliLandingClient initialDemos={[demo]} />);
    expect(html).toContain("Codex CLI");
    expect(html).toContain("OpenAI coding agent");
    expect(html).toContain('href="/try/codex"');
  });

  it("renders demo metadata and an explicit sandbox action without creating a session", () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch");
    const html = renderToStaticMarkup(
      <TryCliDemoClient slug="codex" initialDemo={demo} />,
    );

    expect(html).toContain("codex --version");
    expect(html).toContain("Start sandbox");
    expect(fetchSpy).not.toHaveBeenCalled();
    fetchSpy.mockRestore();
  });
});
