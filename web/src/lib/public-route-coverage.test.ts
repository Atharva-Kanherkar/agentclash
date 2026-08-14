import fs from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";
import {
  PUBLIC_PAGE_ROUTE_COVERAGE,
  REVIEWED_PUBLIC_PAGE_EXCLUSIONS,
  publicPageRouteIsCovered,
} from "./public-route-coverage";

function pageFiles(directory: string): string[] {
  return fs.readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const target = path.join(directory, entry.name);
    if (entry.isDirectory()) return pageFiles(target);
    return entry.name === "page.tsx" ? [target] : [];
  });
}

function routeForPage(filename: string): string {
  const app = path.join(process.cwd(), "src", "app");
  const relative = path.relative(app, path.dirname(filename));
  const segments = relative
    .split(path.sep)
    .filter((segment) => segment && !(segment.startsWith("(") && segment.endsWith(")")));
  return segments.length === 0 ? "/" : `/${segments.join("/")}`;
}

describe("public route coverage manifest", () => {
  it("requires every Next page family to be adapted, private, or explicitly excluded", () => {
    const routes = pageFiles(path.join(process.cwd(), "src", "app")).map(routeForPage);
    for (const route of routes) {
      expect(publicPageRouteIsCovered(route), route).toBe(true);
    }
  });

  it("keeps reviewed exclusions separate from public adapters", () => {
    const adapted = new Set(PUBLIC_PAGE_ROUTE_COVERAGE.map((entry) => entry.route));
    for (const exclusion of REVIEWED_PUBLIC_PAGE_EXCLUSIONS) {
      expect(exclusion.reason.length).toBeGreaterThan(20);
      expect(adapted.has(exclusion.route)).toBe(false);
    }
  });
});
