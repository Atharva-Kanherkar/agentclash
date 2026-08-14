import { afterEach, describe, expect, it, vi } from "vitest";
import { GET as getOpenAPI, HEAD as headOpenAPI } from "@/app/openapi.yaml/route";
import { GET as getSchema, HEAD as headSchema } from "@/app/schemas/[...path]/route";
import { GET as getCliSchema, HEAD as headCliSchema } from "@/app/cli-schema.json/route";

const SCHEMAS = [
  "prompt-eval-result.schema.json",
  "prompt-eval.schema.json",
  "voice-artifact-manifest.schema.json",
  "voice-live-continuity-report.schema.json",
  "voice-source-separation-report.schema.json",
  "voice-video-sync-report.schema.json",
] as const;

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("machine contract routes", () => {
  it("serves the checked OpenAPI source for GET and HEAD", async () => {
    vi.spyOn(console, "log").mockImplementation(() => undefined);
    const request = new Request("https://www.agentclash.dev/openapi.yaml");
    const response = await getOpenAPI(request);
    const body = await response.text();

    expect(response.status).toBe(200);
    expect(response.headers.get("content-type")).toBe(
      "application/yaml; charset=utf-8",
    );
    expect(body).toContain('openapi: "3.1.0"');
    expect(body).toContain("https://api.agentclash.dev");

    const head = await headOpenAPI(request);
    expect(head.status).toBe(200);
    expect(await head.text()).toBe("");
  });

  it.each(SCHEMAS)("serves %s with its deployed canonical ID", async (name) => {
    vi.spyOn(console, "log").mockImplementation(() => undefined);
    const request = new Request(`https://www.agentclash.dev/schemas/${name}`);
    const context = { params: Promise.resolve({ path: [name] }) };
    const response = await getSchema(request, context);
    const body = (await response.json()) as { $id?: string };

    expect(response.status).toBe(200);
    expect(response.headers.get("content-type")).toBe(
      "application/schema+json; charset=utf-8",
    );
    expect(body.$id).toBe(`https://www.agentclash.dev/schemas/${name}`);

    const head = await headSchema(request, context);
    expect(head.status).toBe(200);
    expect(await head.text()).toBe("");
  });

  it("rejects schemas outside the allowlist", async () => {
    vi.spyOn(console, "log").mockImplementation(() => undefined);
    const response = await getSchema(
      new Request("https://www.agentclash.dev/schemas/../private.json"),
      { params: Promise.resolve({ path: ["..", "private.json"] }) },
    );
    expect(response.status).toBe(404);
  });

  it("proxies the released CLI schema with one-hour caching", async () => {
    vi.spyOn(console, "log").mockImplementation(() => undefined);
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation(async () =>
        new Response(
          JSON.stringify({
            schema_version: 1,
            cli_version: "9.8.7",
            commands: [{ name: "version" }],
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      ),
    );

    const response = await getCliSchema(
      new Request("https://www.agentclash.dev/cli-schema.json"),
    );
    const body = (await response.json()) as { cli_version: string };
    expect(response.status).toBe(200);
    expect(body.cli_version).toBe("9.8.7");
    expect(response.headers.get("cache-control")).toContain("s-maxage=3600");
    expect(response.headers.get("x-agentclash-schema-source")).toBe(
      "release-asset",
    );

    const head = await headCliSchema(
      new Request("https://www.agentclash.dev/cli-schema.json"),
    );
    expect(head.status).toBe(200);
    expect(await head.text()).toBe("");
  });

  it("fails honestly when the released CLI asset is unavailable", async () => {
    vi.spyOn(console, "error").mockImplementation(() => undefined);
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response("missing", { status: 404 })),
    );

    const response = await getCliSchema(
      new Request("https://www.agentclash.dev/cli-schema.json"),
    );
    expect(response.status).toBe(503);
    expect(response.headers.get("retry-after")).toBe("300");
  });
});
