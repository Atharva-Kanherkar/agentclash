import { Children, type ReactElement } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { BenchmarkReport } from "@/lib/benchmarks";

const mocks = vi.hoisted(() => ({
  getAllReports: vi.fn(),
  hasPublishedBenchmarks: vi.fn(),
  isReturningVisitor: vi.fn(),
  redirect: vi.fn(),
  withAuth: vi.fn(),
}));

vi.mock("@workos-inc/authkit-nextjs", () => ({
  withAuth: mocks.withAuth,
}));

vi.mock("next/navigation", () => ({
  redirect: mocks.redirect,
}));

vi.mock("@/lib/auth/returning", () => ({
  isReturningVisitor: mocks.isReturningVisitor,
}));

vi.mock("@/lib/benchmarks", () => ({
  getAllReports: mocks.getAllReports,
  hasPublishedBenchmarks: mocks.hasPublishedBenchmarks,
}));

vi.mock("@/lib/changelog", () => ({
  getChangelogPeriods: () => [{ startDate: "2026-08-01" }],
}));

vi.mock("@/components/marketing/json-ld", () => ({
  JsonLd: () => null,
  SITE_URL: "https://www.agentclash.dev",
  faqSchema: () => ({}),
  organizationSchema: () => ({}),
  productSchema: () => ({}),
  softwareSourceCodeSchema: () => ({}),
  websiteSchema: () => ({}),
}));

vi.mock("./landing", () => ({
  default: () => null,
}));

import RootPage from "./page";

function report(): BenchmarkReport {
  return {
    slug: "measured-report",
    title: "Measured benchmark",
    date: "2026-06-07",
    description: "A measured comparison.",
    author: "AgentClash",
    featuredModel: "Model Alpha",
    verdict: "Model Alpha won.",
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
  };
}

function homePageProps(page: ReactElement) {
  const children = Children.toArray(
    (page.props as { children: ReactElement[] }).children,
  );
  return (
    children.at(-1) as ReactElement<{
      benchmark?: { slug: string; results: Array<{ model: string }> };
      returning?: boolean;
    }>
  ).props;
}

describe("homepage benchmark data", () => {
  beforeEach(() => {
    mocks.getAllReports.mockReset();
    mocks.hasPublishedBenchmarks.mockReset();
    mocks.isReturningVisitor.mockReset();
    mocks.redirect.mockReset();
    mocks.withAuth.mockReset();
    mocks.withAuth.mockResolvedValue({ user: null });
    mocks.isReturningVisitor.mockResolvedValue(false);
  });

  it("passes the measured report from the server registry into HomePage", async () => {
    mocks.hasPublishedBenchmarks.mockReturnValue(true);
    mocks.getAllReports.mockReturnValue([report()]);

    const props = homePageProps((await RootPage()) as ReactElement);

    expect(props.returning).toBe(false);
    expect(props.benchmark).toMatchObject({
      slug: "measured-report",
      results: [{ model: "Model Alpha" }],
    });
  });

  it("passes no benchmark and skips the report scan when none are published", async () => {
    mocks.hasPublishedBenchmarks.mockReturnValue(false);

    const props = homePageProps((await RootPage()) as ReactElement);

    expect(props.benchmark).toBeUndefined();
    expect(mocks.getAllReports).not.toHaveBeenCalled();
  });
});
