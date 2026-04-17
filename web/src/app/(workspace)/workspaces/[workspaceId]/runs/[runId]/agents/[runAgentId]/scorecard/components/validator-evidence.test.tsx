import { describe, expect, it } from "vitest";
import {
  buildRegexHighlightSegments,
  prettyEvidenceValue,
} from "./validator-evidence-utils";

describe("buildRegexHighlightSegments", () => {
  it("marks the first regex match when the pattern is valid", () => {
    const segments = buildRegexHighlightSegments("Par[a-z]+", "Paris, France");
    expect(segments).toEqual([
      { text: "Paris", matched: true },
      { text: ", France", matched: false },
    ]);
  });

  it("falls back to unhighlighted text when the pattern is invalid", () => {
    expect(buildRegexHighlightSegments("[", "Paris")).toEqual([
      { text: "Paris", matched: false },
    ]);
  });
});

describe("prettyEvidenceValue", () => {
  it("pretty-prints objects for evidence blocks", () => {
    expect(prettyEvidenceValue({ stdout: "ok", passed_tests: 3 })).toContain(
      "\"stdout\": \"ok\"",
    );
  });

  it("returns an em dash for empty values", () => {
    expect(prettyEvidenceValue(null)).toBe("—");
  });
});
