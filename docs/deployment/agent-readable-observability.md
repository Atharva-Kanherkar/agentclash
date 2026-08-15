# Agent-readable request observability

AgentClash measures crawler and agent access from server-side request data, not
PostHog or other browser analytics. The first rollout uses Vercel Runtime Logs
and the backend API request logger. An external Log Drain is intentionally
deferred.

## Emitted events

The web routing middleware emits `agent_readable_request` for known AI crawler
families, explicit Markdown requests, and machine-contract routes. Node route
handlers emit `agent_readable_response` when they can report status, response
bytes, and duration. The backend request logger enriches public publication API
requests with the same representation and crawler classifications.

Allowlisted fields are:

- normalized pathname without a query string
- HTTP method and status
- known user-agent family, never the full user-agent string
- accept class and requested/served representation
- request ID and route kind
- response bytes and duration where the handler can measure them
- Vercel platform fields such as cache state, deployment ID, and environment

The application logs never include cookies, authorization values, capability
tokens, raw query strings, IP addresses, submitted content, or private share
paths. `/share/{token}` is logged only as `/share/{token}`.

## Runtime Logs queries

Open the web project in Vercel, choose **Logs**, set `environment:production`,
and save or retain these searches. Vercel stores recent per-project queries,
and the route, request path, resource, method, status, cache, deployment, and
request ID are available as filters.

| Query name | Runtime Logs filters and search text | Use |
| --- | --- | --- |
| AI-agent volume | Search `agent_readable_request`; method `GET` or `HEAD` | Group by request path or route and inspect `agent_family` in the JSON message. |
| Markdown delivery | Search `agent_readable_request`; then search `served_representation` | Compare direct `/md` requests with canonical requests served as Markdown. |
| Agent 404 guesses | Search `agent_readable_request`; status `404` | Group by Request Path. Known capability paths remain redacted in the application message. |
| Contract failures | Request Path `/llms.txt`, `/llms-full.txt`, `/openapi.yaml`, `/cli-schema.json`, or `/schemas/*`; status `4xx` or `5xx` | Detect broken discovery and contract endpoints. |
| Publication failures | Route `/publications/[id]`, `/md/[[...path]]`, or backend `/public/publications/{publicationID}`; status `4xx` or `5xx` | Confirm fail-closed removals and investigate unexpected serializer failures. |
| Cache verification | Request Path `/md/*`, `/llms*`, `/openapi.yaml`, `/schemas/*`, or `/cli-schema.json` | Group by cache state and verify HIT/MISS/STALE behavior after rollout. |

For repeatable aggregation, the Vercel CLI can export JSON. The structured
application payload is stored in the log message:

```bash
vercel logs --environment production --since 24h \
  --query agent_readable_request --json \
  | jq -r '(.message | fromjson? // empty) | [.agent_family, .path, .served_representation] | @tsv' \
  | sort | uniq -c | sort -nr
```

Use Vercel request IDs to correlate a middleware request event, a route-handler
response event, and the platform request record. The platform record supplies
status, cache behavior, deployment, and runtime timing when application code
cannot measure them directly.

## Rollout checks

1. Enable `MARKDOWN_NEGOTIATION_ENABLED=true` in Preview only.
2. Request one canonical page with HTML, direct Markdown, negotiated Markdown,
   wildcard Accept, and `text/markdown;q=0`.
3. Verify `Vary`, `Link`, `Content-Location`, robots, and cache headers in the
   response and confirm the three structured events in Runtime Logs.
4. Confirm query strings and capability tokens are absent from log messages.
5. Enable the flag in Production and retain the queries above for the first
   rollout window.

Vercel's Runtime Logs documentation is the source of truth for available
filters and retention: <https://vercel.com/docs/logs/runtime>.
