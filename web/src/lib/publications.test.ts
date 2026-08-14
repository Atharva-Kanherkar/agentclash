import { describe, expect, it } from "vitest";
import type { PublicPublicationResponse, PublicShareResourceType } from "@/lib/api/types";
import {
  publicationTitle,
  renderPublicationCatalogMarkdown,
  renderPublicationMarkdown,
} from "./publications";

function fixture(
  type: PublicShareResourceType,
  resource: Record<string, unknown>,
): PublicPublicationResponse {
  return {
    publication: {
      id: "11111111-1111-4111-8111-111111111111",
      resource_type: type,
      created_at: "2026-08-14T00:00:00Z",
      updated_at: "2026-08-14T01:00:00Z",
      canonical_path: "/publications/11111111-1111-4111-8111-111111111111",
    },
    resource: { type, ...resource },
  };
}

describe("publication renderers", () => {
  it.each([
    fixture("challenge_pack_version", {
      pack: { name: "Support checks", family: "support", description: "Redacted pack" },
      version: { version_number: 2, lifecycle_status: "runnable", manifest: { tasks: 4 } },
    }),
    fixture("run_scorecard", {
      run: { name: "Support race", status: "completed", execution_mode: "comparison" },
      agents: [{ id: "agent-1", label: "Careful", status: "completed" }],
      agent_scorecards: [{ run_agent_id: "agent-1", overall_score: 0.91, passed: true }],
    }),
    fixture("run_agent_scorecard", {
      run: { name: "Support race" },
      run_agent: { label: "Careful", status: "completed" },
      scorecard: { overall_score: 0.91, correctness_score: 0.95, passed: true },
    }),
    fixture("run_agent_replay", {
      run: { name: "Support race" },
      run_agent: { label: "Careful", status: "completed" },
      replay: { event_count: 12, summary: { terminal_state: "complete" } },
    }),
    fixture("agent_tryout", {
      template_slug: "meeting-minutes",
      status: "completed",
      redaction_status: "passed",
      input_snapshot: { notes: "Public notes" },
      summary: { result: "Public result" },
    }),
  ])("renders an allowlisted $resource.type publication", (publication) => {
    const markdown = renderPublicationMarkdown(publication, "https://example.test");
    expect(markdown).toMatch(/^#\s+\S/m);
    expect(markdown).toContain(
      "Source: https://example.test/publications/11111111-1111-4111-8111-111111111111",
    );
    expect(markdown).toContain("User-published content");
    expect(markdown).not.toContain("capability-token");
    expect(publicationTitle(publication)).not.toBe("");
  });

  it("keeps user content inside escaped text or indented JSON blocks", () => {
    const publication = fixture("agent_tryout", {
      template_slug: "# fake heading",
      status: "completed",
      redaction_status: "passed",
      input_snapshot: { notes: "</pre><script>alert(1)</script>\n# injected" },
      summary: {},
    });
    const markdown = renderPublicationMarkdown(publication);

    expect(markdown).toContain("\\# fake heading tryout");
    expect(markdown).toContain('    "notes": "</pre><script>alert(1)</script>\\n# injected"');
    expect(markdown).not.toContain("\n# injected\n");
  });

  it("renders the live catalog as Markdown without capability URLs", () => {
    const publication = fixture("run_scorecard", {
      run: { name: "Support race", status: "completed" },
      agents: [],
      agent_scorecards: [],
    });
    const markdown = renderPublicationCatalogMarkdown(
      [publication],
      "https://example.test",
    );

    expect(markdown).toContain("# Published Agent Evaluation Artifacts");
    expect(markdown).toContain(
      "https://example.test/md/publications/11111111-1111-4111-8111-111111111111",
    );
    expect(markdown).not.toContain("/share/");
  });
});
