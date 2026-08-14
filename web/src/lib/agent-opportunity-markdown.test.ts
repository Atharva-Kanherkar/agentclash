import { describe, expect, it } from "vitest";
import { escapeMarkdownText, renderAgentOpportunityMarkdown } from "./agent-opportunity-markdown";

describe("agent opportunity Markdown", () => {
  it("escapes report content that could change document structure", () => {
    expect(escapeMarkdownText("# injected\n<script>alert(1)</script> | value")).toBe(
      "\\# injected &lt;script&gt;alert\\(1\\)&lt;/script&gt; \\| value",
    );
  });

  it("renders a portable report with fixed semantic sections", () => {
    const markdown = renderAgentOpportunityMarkdown({
      analyzedUrl: "https://example.com",
      companyName: "Example",
      generatedAt: "2026-08-14T00:00:00.000Z",
      agentFitScore: 72,
      fitLevel: "high",
      shouldBuildAgent: "narrow_pilot",
      honestVerdict: "Pilot support triage.",
      summary: "A repeatable workflow exists.",
      useCases: [{
        title: "Support triage",
        workflow: "Classify tickets.",
        fit: "high",
        estimatedMonthlyHoursSaved: "20",
        estimatedMonthlySavingsUsd: "$2,000",
        complexity: "medium",
        why: "High volume.",
        firstEvalTasks: ["Refund escalation"],
      }],
      risks: [{ risk: "Wrong policy", severity: "high", mitigation: "Require review." }],
      evaluationPack: {
        name: "Support",
        recommendedCases: 20,
        adversarialCases: 5,
        successCriteria: ["No policy hallucinations"],
      },
      nextSteps: ["Collect tickets"],
      evidenceLimitations: ["Public pages only"],
    });

    expect(markdown).toContain("# Example agent opportunity report");
    expect(markdown).toContain("## Candidate workflows");
    expect(markdown).toContain("## Evidence limitations");
  });
});
