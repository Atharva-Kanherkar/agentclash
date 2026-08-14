import {
  NextResponse,
  type NextFetchEvent,
  type NextRequest,
} from "next/server";
import { authkitMiddleware } from "@workos-inc/authkit-nextjs";
import {
  CANONICAL_PATH_HEADER,
  NEGOTIATED_MARKDOWN_HEADER,
  agentRequestLog,
  isMarkdownNegotiablePath,
  markdownPathForCanonical,
  representationLinkHeader,
  shouldLogAgentRequest,
  shouldNegotiateMarkdown,
} from "@/lib/public-http";

const authkit = authkitMiddleware();
const PUBLIC_ORIGIN = "https://www.agentclash.dev";

const WORKSPACE_ROOT_PATTERN = /^\/workspaces\/[^/]+\/?$/;

function addVaryAccept(response: Response) {
  const existing = response.headers.get("Vary");
  const values = new Set(
    (existing ?? "")
      .split(",")
      .map((value) => value.trim())
      .filter(Boolean),
  );
  values.add("Accept");
  response.headers.set("Vary", [...values].join(", "));
}

function addRepresentationHeaders<T extends Response>(response: T, pathname: string): T {
  addVaryAccept(response);
  response.headers.set("Link", representationLinkHeader(pathname, PUBLIC_ORIGIN));
  return response;
}

function logAgentReadableRequest(
  request: NextRequest,
  servedRepresentation: "html" | "markdown" | "machine",
) {
  const pathname = request.nextUrl.pathname;
  const accept = request.headers.get("accept");
  const userAgent = request.headers.get("user-agent");
  if (!shouldLogAgentRequest({ pathname, accept, userAgent })) return;
  console.log(
    JSON.stringify(
      agentRequestLog({
        pathname,
        method: request.method,
        accept,
        userAgent,
        requestId: request.headers.get("x-vercel-id"),
        servedRepresentation,
      }),
    ),
  );
}

export default async function middleware(
  request: NextRequest,
  event: NextFetchEvent,
) {
  const pathname = request.nextUrl.pathname;
  const negotiationEnabled =
    process.env.MARKDOWN_NEGOTIATION_ENABLED?.toLowerCase() === "true";

  if (
    shouldNegotiateMarkdown({
      enabled: negotiationEnabled,
      method: request.method,
      pathname,
      accept: request.headers.get("accept"),
    })
  ) {
    const url = request.nextUrl.clone();
    url.pathname = markdownPathForCanonical(pathname);
    url.search = "";
    const requestHeaders = new Headers(request.headers);
    requestHeaders.set(NEGOTIATED_MARKDOWN_HEADER, "1");
    requestHeaders.set(CANONICAL_PATH_HEADER, pathname);
    const response = NextResponse.rewrite(url, {
      request: { headers: requestHeaders },
    });
    logAgentReadableRequest(request, "markdown");
    return addRepresentationHeaders(response, pathname);
  }

  if (
    pathname === "/docs" ||
    pathname.startsWith("/docs/") ||
    pathname === "/docs-md" ||
    pathname.startsWith("/docs-md/") ||
    pathname === "/md" ||
    pathname.startsWith("/md/") ||
    pathname === "/llms.txt" ||
    pathname === "/llms-full.txt" ||
    pathname === "/openapi.yaml" ||
    pathname === "/cli-schema.json" ||
    pathname === "/schemas" ||
    pathname.startsWith("/schemas/") ||
    pathname === "/publications" ||
    pathname.startsWith("/publications/") ||
    pathname === "/share" ||
    pathname.startsWith("/share/")
  ) {
    const response = NextResponse.next();
    logAgentReadableRequest(request, pathname.startsWith("/md") ? "markdown" : "machine");
    return isMarkdownNegotiablePath(pathname)
      ? addRepresentationHeaders(response, pathname)
      : response;
  }

  if (WORKSPACE_ROOT_PATTERN.test(pathname)) {
    const url = request.nextUrl.clone();
    url.pathname = `${pathname.replace(/\/$/, "")}/runs`;
    return NextResponse.redirect(url, 307);
  }

  const response = (await authkit(request, event)) ?? NextResponse.next();
  if (isMarkdownNegotiablePath(pathname)) {
    logAgentReadableRequest(request, "html");
    return addRepresentationHeaders(response, pathname);
  }
  return response;
}

export const config = {
  matcher: [
    "/((?!_next/static|_next/image|favicon.ico|og-image.png|twitter-image.png|favicon-16x16.png|favicon-32x32.png|favicon-96x96.png|apple-touch-icon.png|robots.txt|sitemap.xml|265a46be97a2ce1f8891dd452d243327.txt).*)",
  ],
};
