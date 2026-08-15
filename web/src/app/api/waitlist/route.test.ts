import { describe, expect, it } from "vitest";
import { POST } from "./route";

describe("waitlist native form fallback", () => {
  it("redirects an invalid native form submission without requiring JavaScript", async () => {
    const response = await POST(
      new Request("https://www.agentclash.dev/api/waitlist", {
        method: "POST",
        headers: { "content-type": "application/x-www-form-urlencoded" },
        body: new URLSearchParams({
          email: "not-an-email",
          source: "eval-checklist",
          resource: "eval-checklist",
          intent: "resource-download",
        }),
      }),
    );

    expect(response.status).toBe(303);
    expect(response.headers.get("location")).toContain(
      "/resources/eval-checklist?form_error=Enter+a+valid+email.",
    );
  });
});
