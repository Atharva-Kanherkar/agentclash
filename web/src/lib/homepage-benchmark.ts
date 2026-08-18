import type { BenchmarkReport, BenchmarkResult } from "./benchmarks";

export type HomepageBenchmarkData = {
  slug: string;
  title: string;
  verdict: string;
  challengePack: string;
  featuredModel: string;
  results: BenchmarkResult[];
};

/**
 * Select the newest measured report for the homepage. `getAllReports()` returns
 * reports newest-first, so the first non-sample entry is the one to promote.
 */
export function selectHomepageBenchmark(
  reports: readonly BenchmarkReport[],
): HomepageBenchmarkData | undefined {
  const report = reports.find((candidate) => !candidate.sample);
  if (!report || report.results.length === 0) return undefined;

  return {
    slug: report.slug,
    title: report.title,
    verdict: report.verdict,
    challengePack: report.challengePack,
    featuredModel: report.featuredModel,
    results: report.results,
  };
}
