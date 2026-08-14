# codex/agent-readable-web - Test Contract

## Functional Behavior

- Every indexable public URL represented by the public-content registry resolves to semantic HTML and non-empty Markdown.
- `GET /md/{canonical-path}` returns the explicit Markdown representation. `GET /md` represents the homepage.
- Existing `GET /docs-md/*` URLs continue to return the same document content through the shared renderer.
- Canonical public URLs return HTML by default. They return Markdown only for public `GET` or `HEAD` requests whose exact `text/markdown` quality is positive and strictly preferred over exact `text/html`.
- Missing `Accept`, wildcard-only ranges, media-type wildcards, ties, and `text/markdown;q=0` return HTML.
- Negotiation never rewrites private/authenticated, API, ingestion, framework-internal, token-share, or mutation requests and never forwards query parameters to Markdown renderers.
- Eligible HTML and Markdown responses advertise canonical and Markdown representations through `Link` and `Vary: Accept` headers.
- Direct `/md` responses are `noindex, follow`; negotiated Markdown at the canonical URL is not marked `noindex`.
- `/llms.txt`, `/llms-full.txt`, sitemap entries, docs search, and IndexNow discovery use the shared public-content registry.
- `/llms.txt` links to exact Markdown and machine-contract URLs. `/llms-full.txt` excludes user-generated publication bodies.
- `/openapi.yaml`, `/schemas/*.json`, and `/cli-schema.json` expose the supported machine contracts with correct content types and canonical production URLs.
- Generic `/share/{token}` pages remain disallowed and always noindex, including when the share has `search_indexing=true`.
- `/publications` and `/publications/{id}` expose only active, unexpired, search-indexing-enabled resources serialized through supported redacted public renderers.
- Publication lookups use the non-secret share record ID, never the capability token. Ineligible and unsupported resources return `404` without existence disclosure.
- Publication Markdown labels and structurally escapes user-supplied content. It never contains capability tokens, credentials, environment values, or private user data.
- `/try` and `/try/{slug}` include catalog/detail content in the server response. A GET never creates or resumes a sandbox; that mutation requires an explicit user action.
- `/tryouts` server-renders public templates, tools, selected public inputs, and completed public results when available. Query-based sessions remain noindex.
- `/agent-opportunity` documents its POST contract in Markdown. Generation remains POST-only and transient output is not indexable without explicit publication.
- Pricing HTML and Markdown include both monthly and annual values from one pricing source.
- Agent request measurement uses structured server logs, not product-analytics events, and omits query strings, cookies, authorization data, tokens, IP addresses, and submitted content.
- The canonical public host is `https://www.agentclash.dev`.

## Unit Tests

- Public-path normalization accepts canonical paths and rejects traversal, malformed encoding, duplicate separators, private prefixes, and unknown paths.
- The content registry resolves every supported static and dynamic route family and produces an H1, canonical source, and semantic body.
- Registry inclusion flags drive llms, full-bundle, search, sitemap, and IndexNow behavior consistently.
- The `Accept` parser covers missing headers, exact Markdown, exact HTML, wildcards, quality values, ties, invalid quality values, and `q=0`.
- Negotiation classification covers public/private paths, `GET`, `HEAD`, mutation methods, explicit `/md`, `/docs-md`, machine contracts, and query stripping.
- Markdown response headers distinguish direct from internally negotiated responses.
- Markdown escaping neutralizes raw HTML and delimiter injection while retaining readable user content.
- Agent user-agent classification covers every named crawler allowed by robots plus unknown/generic clients.
- Structured log payloads contain only allowlisted fields and normalized pathnames.
- Publication eligibility covers active, expired, revoked, indexing-disabled, unsupported, and malformed shares.
- Every supported public-share resource type has a redacted semantic serializer, including agent tryouts.

## Integration / Functional Tests

- Every link emitted by `/llms.txt` resolves with the declared content type.
- `/llms-full.txt` renders all included static content and contains no publication body or share token.
- The canonical sitemap contains registry-backed HTML URLs only, with real modification dates, and excludes `/md`, `/docs-md`, machine artifacts, token shares, and private routes.
- HTML metadata and response headers advertise the matching `/md` alternate without overwriting existing RSS alternates.
- Direct and negotiated Markdown contain equivalent titles, key facts, tables, pricing periods, and important links to the HTML source.
- OpenAPI lists `https://api.agentclash.dev`; every published JSON Schema has a resolvable canonical `$id` under `https://www.agentclash.dev/schemas/`.
- The release workflow generates a parseable CLI schema whose `cli_version` matches the release tag before upload.
- Publication list/detail APIs and HTML/Markdown routes return identical redacted semantics and immediately return `404` after revocation, expiry, or indexing opt-out.
- The existing `/docs-md` route and new `/md/docs` route share one renderer and stay content-equivalent.

## Smoke Tests

- `pnpm lint`, `pnpm test`, and `pnpm build` succeed under `web/`.
- `go test -short -race -count=1 ./...` succeeds under `backend/` and `cli/` for affected packages.
- A production-mode server returns 200 for `/llms.txt`, `/llms-full.txt`, `/md`, `/md/docs`, `/openapi.yaml`, one JSON Schema, and `/cli-schema.json`.
- A canonical public page returns HTML for normal browser headers and Markdown for an explicit preferred `text/markdown` header when negotiation is enabled.
- Plain HTTP responses for `/try`, one demo detail, and `/tryouts` contain meaningful catalog/template text before JavaScript executes.

## E2E Tests

- Visit `/try/{slug}` without clicking the start control and verify no sandbox-creation request occurs; click the control and verify exactly one creation/resume request occurs.
- Open a valid opted-in publication in HTML and Markdown, then revoke or disable indexing and verify both URLs return `404`.
- Fetch a private route with `Accept: text/markdown` and verify it is not rewritten or exposed.

## Manual / cURL Tests

Run against a local production server with negotiation enabled:

```bash
curl -i http://localhost:3000/pricing
curl -i -H 'Accept: text/markdown' http://localhost:3000/pricing
curl -i -H 'Accept: text/html, text/markdown;q=0.5' http://localhost:3000/pricing
curl -i -H 'Accept: */*' http://localhost:3000/pricing
curl -I -H 'Accept: text/markdown' http://localhost:3000/docs
curl -i http://localhost:3000/md/pricing
curl -i http://localhost:3000/docs-md
curl -i http://localhost:3000/llms.txt
curl -i http://localhost:3000/openapi.yaml
curl -i http://localhost:3000/cli-schema.json
curl -i -H 'Accept: text/markdown' http://localhost:3000/workspaces
```

Publication checks require a known eligible and ineligible share ID:

```bash
curl -i http://localhost:3000/publications/PUBLIC_SHARE_ID
curl -i http://localhost:3000/md/publications/PUBLIC_SHARE_ID
curl -i http://localhost:8080/public/publications/PUBLIC_SHARE_ID
curl -i http://localhost:8080/public/publications/INELIGIBLE_SHARE_ID
```
