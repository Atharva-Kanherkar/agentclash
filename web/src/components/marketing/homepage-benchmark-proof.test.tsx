import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import Link from "next/link";

import type { HomepageBenchmarkData } from "@/lib/homepage-benchmark";
import { HomepageBenchmarkProof } from "./homepage-benchmark-proof";

const benchmark: HomepageBenchmarkData = {
  slug: "measured-report",
  title: "Four models on an expression evaluator",
  verdict: "Model Alpha completed the task with fewer turns.",
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
    {
      model: "Model Beta",
      provider: "Provider",
      rank: 2,
      winner: false,
      composite: 0.75,
      correctness: 0.8,
      reliability: 0.7,
      latency: null,
      cost: 0.6,
      costPerCorrectUsd: null,
    },
  ],
};

describe("HomepageBenchmarkProof", () => {
  it("renders measured report proof with the shared scoreboard and links", () => {
    const html = renderToStaticMarkup(
      <HomepageBenchmarkProof
        benchmark={benchmark}
        actions={
          <>
            <Link href="/benchmarks/measured-report">Reproduce this run</Link>
            <Link href="/benchmarks">Browse all benchmarks</Link>
          </>
        }
      />,
    );

    expect(html).toContain("Measured benchmark");
    expect(html).toContain("A real run. A result you can reproduce.");
    expect(html).toContain("Four models on an expression evaluator");
    expect(html).toContain("Model Alpha completed the task with fewer turns.");
    expect(html).toContain("Expression Evaluator Arena v1");
    expect(html).toContain("Model Alpha");
    expect(html).toContain("Model Beta");
    expect(html).toContain("Composite");
    expect(html).toContain('href="/benchmarks/measured-report"');
    expect(html).toContain("Reproduce this run");
    expect(html).toContain('href="/benchmarks"');
    expect(html).toContain("Browse all benchmarks");
  });

  it("renders nothing when there is no measured benchmark", () => {
    expect(
      renderToStaticMarkup(<HomepageBenchmarkProof benchmark={undefined} />),
    ).toBe("");
  });
});
