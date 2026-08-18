import { describe, expect, it } from "vitest";

import type { BenchmarkReport } from "./benchmarks";
import { selectHomepageBenchmark } from "./homepage-benchmark";

function report(overrides: Partial<BenchmarkReport> = {}): BenchmarkReport {
  return {
    slug: "measured-report",
    title: "Measured benchmark",
    date: "2026-06-07",
    description: "A measured comparison.",
    author: "AgentClash",
    featuredModel: "Model Alpha",
    verdict: "Model Alpha won with fewer turns.",
    challengePack: "Expression Evaluator Arena v1",
    sample: false,
    runShareUrl: "",
    tasks: [],
    results: [
      {
        model: "Model Alpha",
        provider: "Provider",
        rank: 1,
        winner: true,
        composite: 1,
        correctness: 1,
        reliability: 1,
        latency: null,
        cost: 0.8,
        costPerCorrectUsd: null,
      },
    ],
    ...overrides,
  };
}

describe("selectHomepageBenchmark", () => {
  it("selects the first non-sample report from the date-sorted registry", () => {
    const selected = selectHomepageBenchmark([
      report({ slug: "newer-sample", sample: true }),
      report({ slug: "newest-real", title: "Newest real report" }),
      report({ slug: "older-real", title: "Older real report" }),
    ]);

    expect(selected).toMatchObject({
      slug: "newest-real",
      title: "Newest real report",
      featuredModel: "Model Alpha",
    });
  });

  it("returns undefined when every report is illustrative", () => {
    expect(
      selectHomepageBenchmark([
        report({ slug: "sample-a", sample: true }),
        report({ slug: "sample-b", sample: true }),
      ]),
    ).toBeUndefined();
  });

  it("returns undefined when the newest real report has no result rows", () => {
    expect(
      selectHomepageBenchmark([report({ results: [] })]),
    ).toBeUndefined();
  });

  it("returns only fields needed by the client-side landing section", () => {
    expect(selectHomepageBenchmark([report()])).toEqual({
      slug: "measured-report",
      title: "Measured benchmark",
      verdict: "Model Alpha won with fewer turns.",
      challengePack: "Expression Evaluator Arena v1",
      featuredModel: "Model Alpha",
      results: [
        {
          model: "Model Alpha",
          provider: "Provider",
          rank: 1,
          winner: true,
          composite: 1,
          correctness: 1,
          reliability: 1,
          latency: null,
          cost: 0.8,
          costPerCorrectUsd: null,
        },
      ],
    });
  });
});
