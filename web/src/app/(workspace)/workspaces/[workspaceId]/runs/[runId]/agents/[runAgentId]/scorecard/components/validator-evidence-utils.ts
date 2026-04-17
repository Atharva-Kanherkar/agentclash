export type RegexHighlightSegment = {
  text: string;
  matched: boolean;
};

export function buildRegexHighlightSegments(
  pattern?: string,
  actual?: string,
): RegexHighlightSegment[] | null {
  if (!pattern || actual == null || actual === "") {
    return actual == null ? null : [{ text: actual, matched: false }];
  }

  try {
    const regex = new RegExp(pattern);
    const match = regex.exec(actual);
    if (!match || match[0] === "") {
      return [{ text: actual, matched: false }];
    }

    const start = match.index ?? actual.indexOf(match[0]);
    const end = start + match[0].length;
    const segments: RegexHighlightSegment[] = [];
    if (start > 0) {
      segments.push({ text: actual.slice(0, start), matched: false });
    }
    segments.push({ text: actual.slice(start, end), matched: true });
    if (end < actual.length) {
      segments.push({ text: actual.slice(end), matched: false });
    }
    return segments;
  } catch {
    return [{ text: actual, matched: false }];
  }
}

export function prettyEvidenceValue(value: unknown): string {
  if (value == null) {
    return "—";
  }
  if (typeof value === "string") {
    return value;
  }
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}
