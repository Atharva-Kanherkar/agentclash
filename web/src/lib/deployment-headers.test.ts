import fs from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";

describe("Vercel representation cache headers", () => {
  it("varies deployed responses on Accept without dropping Next router keys", () => {
    const config = JSON.parse(
      fs.readFileSync(path.join(process.cwd(), "vercel.json"), "utf8"),
    ) as {
      headers?: Array<{
        source: string;
        headers: Array<{ key: string; value: string }>;
      }>;
    };
    const globalRule = config.headers?.find((rule) => rule.source === "/(.*)");
    const vary = globalRule?.headers.find(
      (header) => header.key.toLowerCase() === "vary",
    )?.value;

    expect(vary?.split(/,\s*/)).toEqual(
      expect.arrayContaining([
        "Accept",
        "Accept-Encoding",
        "RSC",
        "Next-Router-State-Tree",
        "Next-Router-Prefetch",
        "Next-Router-Segment-Prefetch",
      ]),
    );
  });
});
