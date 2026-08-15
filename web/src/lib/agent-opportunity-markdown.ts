import type { AgentOpportunityReport } from "@/lib/agent-opportunity";

export function escapeMarkdownText(value: string): string {
  return value
    .replace(/\\/g, "\\\\")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/([`*_{}\[\]()#+.!|~-])/g, "\\$1")
    .replace(/[\r\n]+/g, " ")
    .trim();
}

export function renderAgentOpportunityMarkdown(
  report: AgentOpportunityReport,
): string {
  const text = escapeMarkdownText;
  const lines = [
    `# ${text(report.companyName)} agent opportunity report`,
    "",
    `Analyzed URL: ${text(report.analyzedUrl)}`,
    `Generated: ${text(report.generatedAt)}`,
    `Agent fit: ${report.agentFitScore}/100 (${text(report.fitLevel)})`,
    `Recommendation: ${text(report.shouldBuildAgent)}`,
    "",
    text(report.summary),
    "",
    "## Verdict",
    "",
    text(report.honestVerdict),
    "",
    "## Candidate workflows",
  ];

  for (const useCase of report.useCases) {
    lines.push(
      "",
      `### ${text(useCase.title)}`,
      "",
      text(useCase.workflow),
      "",
      `- Fit: ${text(useCase.fit)}`,
      `- Complexity: ${text(useCase.complexity)}`,
      `- Estimated monthly hours saved: ${text(useCase.estimatedMonthlyHoursSaved)}`,
      `- Estimated monthly savings: ${text(useCase.estimatedMonthlySavingsUsd)}`,
      `- Rationale: ${text(useCase.why)}`,
      ...useCase.firstEvalTasks.map((task) => `- First eval task: ${text(task)}`),
    );
  }

  lines.push("", "## Risks", "");
  for (const risk of report.risks) {
    lines.push(
      `- **${text(risk.severity)}**: ${text(risk.risk)}. Mitigation: ${text(risk.mitigation)}`,
    );
  }

  lines.push(
    "",
    "## Evaluation pack",
    "",
    `- Name: ${text(report.evaluationPack.name)}`,
    `- Realistic cases: ${report.evaluationPack.recommendedCases}`,
    `- Adversarial cases: ${report.evaluationPack.adversarialCases}`,
    ...report.evaluationPack.successCriteria.map(
      (criterion) => `- Success criterion: ${text(criterion)}`,
    ),
    "",
    "## Evidence limitations",
    "",
    ...report.evidenceLimitations.map((limit) => `- ${text(limit)}`),
    "",
    "## Next steps",
    "",
    ...report.nextSteps.map((step, index) => `${index + 1}. ${text(step)}`),
  );

  return lines.join("\n").trim();
}
