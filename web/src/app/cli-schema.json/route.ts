export const dynamic = "force-dynamic";

const RELEASE_SCHEMA_URL =
  "https://github.com/agentclash/agentclash/releases/latest/download/cli-schema.json";

type CliSchema = {
  schema_version: number;
  cli_version: string;
  commands: unknown[];
  [key: string]: unknown;
};

function isCliSchema(value: unknown): value is CliSchema {
  if (!value || typeof value !== "object") return false;
  const schema = value as Partial<CliSchema>;
  return (
    typeof schema.schema_version === "number" &&
    typeof schema.cli_version === "string" &&
    Array.isArray(schema.commands)
  );
}

async function releasedSchema(): Promise<{ schema: CliSchema; source: string }> {
  const releaseResponse = await fetch(RELEASE_SCHEMA_URL, {
    headers: { Accept: "application/json" },
    next: { revalidate: 3600 },
  });
  if (releaseResponse.ok) {
    const candidate: unknown = await releaseResponse.json();
    if (isCliSchema(candidate)) return { schema: candidate, source: "release-asset" };
    throw new Error("Latest release CLI schema is invalid");
  }
  throw new Error(
    `Latest release CLI schema returned ${releaseResponse.status}`,
  );
}

async function cliSchemaResponse(request: Request, head: boolean) {
  const startedAt = Date.now();
  try {
    const { schema, source } = await releasedSchema();
    const body = `${JSON.stringify(schema, null, 2)}\n`;
    console.log(
      JSON.stringify({
        level: "info",
        event: "agent_readable_response",
        path: "/cli-schema.json",
        representation: "machine",
        status: 200,
        bytes: Buffer.byteLength(body, "utf8"),
        duration_ms: Date.now() - startedAt,
        request_id: request.headers.get("x-vercel-id"),
        contract_source: source,
      }),
    );
    return new Response(head ? null : body, {
      headers: {
        "Content-Type": "application/json; charset=utf-8",
        "Cache-Control": "public, max-age=0, s-maxage=3600, stale-while-revalidate=604800",
        "X-AgentClash-Schema-Source": source,
        "X-Content-Type-Options": "nosniff",
        "X-Robots-Tag": "noindex, follow",
      },
    });
  } catch (error) {
    console.error(
      JSON.stringify({
        level: "error",
        event: "agent_readable_response",
        path: "/cli-schema.json",
        representation: "machine",
        status: 503,
        duration_ms: Date.now() - startedAt,
        request_id: request.headers.get("x-vercel-id"),
        error: error instanceof Error ? error.message : String(error),
      }),
    );
    return new Response(head ? null : "CLI schema unavailable", {
      status: 503,
      headers: { "Retry-After": "300" },
    });
  }
}

export function GET(request: Request) {
  return cliSchemaResponse(request, false);
}

export function HEAD(request: Request) {
  return cliSchemaResponse(request, true);
}
