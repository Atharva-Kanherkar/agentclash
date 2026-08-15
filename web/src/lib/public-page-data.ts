import { createApiClient } from "@/lib/api/client";
import {
  getPublicAgentTryout,
  getPublicAgentTryoutEvents,
  listAgentTryoutTemplates,
} from "@/lib/api/agent-tryouts";
import { getTryCliApiBase } from "@/lib/try-cli/config";
import type {
  AgentTryout,
  AgentTryoutTemplate,
  TryoutTimelineEvent,
} from "@/lib/api/types";
import type { DemoMeta } from "@/lib/try-cli/types";
import {
  getBundledTryCliDemo,
  getBundledTryCliDemos,
} from "@/lib/try-cli/catalog.server";

export type PublicTryoutTool = {
  slug: string;
  name: string;
  category: string;
  blurb?: string;
};

export type PublicTryoutPageData = {
  templates: AgentTryoutTemplate[];
  tools: PublicTryoutTool[];
  tryout: AgentTryout | null;
  events: TryoutTimelineEvent[];
};

function isDemoMeta(value: unknown): value is DemoMeta {
  if (!value || typeof value !== "object") return false;
  const demo = value as Partial<DemoMeta>;
  return (
    typeof demo.slug === "string" &&
    typeof demo.name === "string" &&
    typeof demo.sessionMinutes === "number" &&
    Array.isArray(demo.commands)
  );
}

async function fetchTryCliJson(path: string): Promise<unknown> {
  const base = getTryCliApiBase().replace(/\/$/, "");
  const response = await fetch(`${base}${path}`, {
    headers: { Accept: "application/json" },
    next: { revalidate: 3600 },
  });
  if (!response.ok) return null;
  return response.json();
}

export async function getPublicTryCliDemos(): Promise<DemoMeta[]> {
  const bundled = getBundledTryCliDemos();
  if (bundled.length > 0) return bundled;
  try {
    const payload = await fetchTryCliJson("/demos");
    return Array.isArray(payload) ? payload.filter(isDemoMeta) : [];
  } catch {
    return [];
  }
}

export async function getPublicTryCliDemo(slug: string): Promise<DemoMeta | null> {
  if (!/^[a-z0-9][a-z0-9-]{0,79}$/.test(slug)) return null;
  const bundled = getBundledTryCliDemo(slug);
  if (bundled) return bundled;
  try {
    const payload = await fetchTryCliJson(`/demos/${encodeURIComponent(slug)}`);
    return isDemoMeta(payload) ? payload : null;
  } catch {
    return null;
  }
}

function shortToolDescription(description?: string): string | undefined {
  if (!description) return undefined;
  const first = description.split(/[.\n]/)[0]?.trim();
  if (!first) return undefined;
  return first.length > 48 ? `${first.slice(0, 46)}…` : first;
}

export async function getPublicTryoutPageData(
  tryoutId?: string,
): Promise<PublicTryoutPageData> {
  const api = createApiClient();
  const [templateResult, toolResult, tryoutResult, eventsResult] = await Promise.allSettled([
    listAgentTryoutTemplates(api),
    api.get<{
      items: Array<{
        slug: string;
        name: string;
        category?: string;
        description?: string;
      }>;
    }>("/v1/tool-library"),
    tryoutId ? getPublicAgentTryout(api, tryoutId) : Promise.resolve(null),
    tryoutId
      ? getPublicAgentTryoutEvents(api, tryoutId, { after: 0, limit: 200 })
      : Promise.resolve(null),
  ]);

  const templates =
    templateResult.status === "fulfilled"
      ? templateResult.value.items.filter(
          (template) => template.available && template.anonymous_enabled,
        )
      : [];
  const tools =
    toolResult.status === "fulfilled"
      ? (toolResult.value.items ?? []).map((tool) => ({
          slug: tool.slug,
          name: tool.name,
          category: tool.category || "Tools",
          blurb: shortToolDescription(tool.description),
        }))
      : [];

  return {
    templates,
    tools,
    tryout:
      tryoutResult.status === "fulfilled" ? tryoutResult.value : null,
    events:
      eventsResult.status === "fulfilled" && eventsResult.value
        ? eventsResult.value.events
        : [],
  };
}
