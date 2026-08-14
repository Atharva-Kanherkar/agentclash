import { describe, expect, it, vi } from "vitest";
import { GET as getIndex, HEAD as headIndex } from "@/app/llms.txt/route";
import { GET as getFull, HEAD as headFull } from "@/app/llms-full.txt/route";

describe("LLM discovery routes", () => {
  it.each([
    ["/llms.txt", getIndex, headIndex],
    ["/llms-full.txt", getFull, headFull],
  ] as const)("serves GET and HEAD for %s", async (pathname, get, head) => {
    vi.spyOn(console, "log").mockImplementation(() => undefined);
    const request = new Request(`https://www.agentclash.dev${pathname}`);

    const getResponse = get(request);
    expect(getResponse.status).toBe(200);
    expect(getResponse.headers.get("content-type")).toBe(
      "text/plain; charset=utf-8",
    );
    expect((await getResponse.text()).length).toBeGreaterThan(100);

    const headResponse = head(request);
    expect(headResponse.status).toBe(200);
    expect(headResponse.headers.get("content-type")).toBe(
      "text/plain; charset=utf-8",
    );
    expect(await headResponse.text()).toBe("");
  });

  it("advertises only canonical-host HTTPS links", async () => {
    vi.spyOn(console, "log").mockImplementation(() => undefined);
    const response = getIndex(
      new Request("https://www.agentclash.dev/llms.txt"),
    );
    const body = await response.text();
    const links = [...body.matchAll(/\]\((https:\/\/[^)]+)\)/g)].map(
      (match) => new URL(match[1]),
    );

    expect(links.length).toBeGreaterThan(10);
    expect(links.every((url) => url.host === "www.agentclash.dev")).toBe(true);
    expect(body).not.toContain("https://agentclash.dev");
  });
});
