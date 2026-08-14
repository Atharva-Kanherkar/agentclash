import { beforeEach, describe, expect, it, vi } from "vitest";

const withAuthMock = vi.hoisted(() => vi.fn());
const submitIndexNowMock = vi.hoisted(() => vi.fn());

vi.mock("@workos-inc/authkit-nextjs", () => ({
  withAuth: withAuthMock,
}));

vi.mock("@/lib/indexnow", () => ({
  submitIndexNow: submitIndexNowMock,
}));

import { POST } from "./route";

const PUBLICATION_ID = "11111111-1111-4111-8111-111111111111";

describe("POST /api/indexnow/publication", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    withAuthMock.mockResolvedValue({ user: { id: "user-1" } });
    submitIndexNowMock.mockResolvedValue({ status: 202, body: "" });
  });

  it("requires an authenticated publisher", async () => {
    withAuthMock.mockResolvedValue({ user: null });
    const response = await POST(
      new Request("https://www.agentclash.dev/api/indexnow/publication", {
        method: "POST",
        body: JSON.stringify({ publication_id: PUBLICATION_ID }),
      }),
    );

    expect(response.status).toBe(401);
    expect(submitIndexNowMock).not.toHaveBeenCalled();
  });

  it("rejects values that are not publication UUIDs", async () => {
    const response = await POST(
      new Request("https://www.agentclash.dev/api/indexnow/publication", {
        method: "POST",
        body: JSON.stringify({ publication_id: "../../private" }),
      }),
    );

    expect(response.status).toBe(400);
    expect(submitIndexNowMock).not.toHaveBeenCalled();
  });

  it("submits only the canonical publication URL", async () => {
    const response = await POST(
      new Request("https://www.agentclash.dev/api/indexnow/publication?token=ignored", {
        method: "POST",
        body: JSON.stringify({ publication_id: PUBLICATION_ID }),
      }),
    );

    expect(response.status).toBe(202);
    expect(submitIndexNowMock).toHaveBeenCalledWith([
      `https://www.agentclash.dev/publications/${PUBLICATION_ID}`,
    ]);
  });
});
