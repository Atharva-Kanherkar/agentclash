import { NextRequest, type NextFetchEvent } from "next/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

const authkitMock = vi.hoisted(() =>
  vi.fn(async () => {
    const { NextResponse } = await import("next/server");
    return NextResponse.next();
  }),
);

vi.mock("@workos-inc/authkit-nextjs", () => ({
  authkitMiddleware: () => authkitMock,
}));

import middleware from "./middleware";

const event = {} as NextFetchEvent;

function request(path: string) {
  return new NextRequest(`https://www.agentclash.dev${path}`);
}

describe("middleware AuthKit coverage", () => {
  beforeEach(() => {
    authkitMock.mockClear();
  });

  it.each(["/", "/docs", "/blog", "/compare", "/publications", "/pricing"])(
    "runs AuthKit on public HTML %s",
    async (path) => {
      await middleware(request(path), event);
      expect(authkitMock).toHaveBeenCalledOnce();
    },
  );

  it("does not run AuthKit on machine markdown", async () => {
    await middleware(request("/md/pricing"), event);
    expect(authkitMock).not.toHaveBeenCalled();
  });
});
