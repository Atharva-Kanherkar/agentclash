# Vibe Evals V1 implementation

Vibe Evals is a private conversation for exploring, building and improving a text agent. A vague brief gets a question; a sufficient brief can produce an editable agent and evaluation. Casual conversation does not get an invented score. Accepted drafts can be tried, checked, improved, retested and saved into the existing workspace builder.

The UI uses the existing Geist fonts, `builder-*` tokens, surfaces and ClashMark. `/vibe-evals` is a public entry route with private authenticated/capability-scoped data and `noindex`. Public and workspace navigation expose it only when `NEXT_PUBLIC_VIBE_EVALS_ENABLED=true`.

## What ships

- Conversation, structured requirements and immutable artifact revisions persist in PostgreSQL. A session URL resumes the same conversation. Chat can propose drafts but cannot call tools or run evaluations itself.
- Three model roles are independent: assistant, target and evaluator. The initial supported catalog is OpenRouter's `openai/gpt-4o-mini`, `openai/gpt-4.1-mini`, and `openai/gpt-4.1`, subject to configured, current price profiles. The trial evaluator is pinned to GPT-4.1 Mini.
- JSON/YAML imports and versioned Vibe exports can be reviewed without executing anything. Unsupported pack capabilities produce an explicit error; no cases or evaluators are silently removed.
- An accepted agent supports a playground and a complete small check. The scorecard counts persisted PASS/FAIL/UNKNOWN results in code, exposes check coverage and loads full evidence per case. Technical failures remain separate from behavioral failures.
- Coaching is constrained to the accepted evaluation blueprint in code. Retests additionally pin the baseline's blueprint and evaluator. The UI compares the deterministic counts before/after; it does not infer a statistical improvement from three examples.
- Save atomically creates a normal `agent_builds` draft/version and an editable `challenge_pack_drafts` composition. Repeated saves of the same revision reuse the draft. A later accepted artifact creates a new build version and pack draft. Evidence stays in the private Vibe session. Saving into a workspace switches subsequent operations to its credits, with that consequence explained before saving.
- An organization wallet supports verified Dodo top-ups and configured subscription allowances. No model has payment authority. Purchases require a current organization administrator and a user click through Dodo checkout.

Saved session URLs and the canonical workspace draft are the return paths in V1. A full conversation-history sidebar/search is deferred.

## Reused architecture and merged work

The base is `439e20ba`, including merged PR #1245 (`8f20f21a`) and #1246 (`49f3dd0c`). Vibe uses #1246's blueprint contract/allowlists and the same `BundleToComposition → ComposeBundle → ValidateBundle` path shared with #1245. It **executes the composed bundle**, where judge modes and required defaults have been resolved. Existing tryout-to-pack promotion remains intact. No tryout rows are fabricated for private sessions.

The legacy builder's fallback that retries without judges is not used. Invalid judge references, removed dimensions, duplicate cases and unsupported execution capabilities are rejected visibly. PRs #1073–#1076 and the discontinued HTTP guide-agent direction are not dependencies.

Orchestration stays in Go and the existing Temporal service. The `vibe-evals` task queue is isolation within Temporal, not a new queue system. PostgreSQL owns operation identity, revision checks, authorization, admission, wallet holds, events, attempts and results. Redis provides distributed admission rate limits and two SSE leases per actor. The existing runtime provider adapter makes OpenRouter calls. The existing scoring engine runs the supported deterministic validators.

Key code:

| Responsibility | Implementation |
| --- | --- |
| HTTP, capability cookies, safe bodies, SSE, case evidence | `backend/internal/api/vibe.go` |
| Strict reuse of pack generation/composition | `backend/internal/api/vibe_compiler.go` |
| Admission, quotes, reservations, settlement | `backend/internal/vibe/admission.go` |
| Provider attempt journal, current authorization, saved results | `backend/internal/vibe/attempts.go` |
| Complete model input counting, prices, role routing | `backend/internal/vibe/models.go`, `gateway.go` |
| Conversation/evaluation execution | `backend/internal/vibe/service.go`, `runner.go`, `judge.go` |
| Durable workflow, outbox, reconciliation, expiry | `backend/internal/vibe/temporal.go` |
| Private data, CAS edits and canonical save | `backend/internal/vibe/store.go`, `requirements.go` |
| Verified credits and refunds/disputes | `backend/internal/api/vibe_credits.go`, `backend/internal/vibe/credits.go` |
| Theme-preserving chat, artifacts and scorecards | `web/src/app/vibe-evals`, `web/src/components/vibe` |

## Hard limits

`backend/internal/vibe/limits.go` is the limits source. A signed-in session without a workspace still uses the trial limits. The JSON mutation envelope and the separate single-file import endpoint have different byte caps; compressed uploads are rejected.

| Limit | Trial | Workspace |
| --- | ---: | ---: |
| JSON HTTP body | 64 KiB | 256 KiB |
| Individual message / agent instructions | 16 KiB | 64 KiB |
| Import file, one per request | 256 KiB | 1 MiB |
| Imports per conversation | 3 | 5 |
| Artifact bytes, including generated revisions | 2 MiB | 20 MiB, also subject to document cap |
| Entire persisted conversation document | 8 MiB | 8 MiB |
| JSON/YAML nesting | 16 | 24 |
| Structured nodes | 10,000 | 50,000 |
| Object keys / array items | 128 / 256 | 256 / 1,024 |
| String bytes | 16 KiB | 64 KiB |
| Object, case and check key bytes | 128 | 128 |
| Number literal bytes | 64 | 64 |
| Imported cases | 50 | 200 |
| Executed cases per run | 3 | 20 |
| Evaluators per case | 1 | 2 |
| Deterministic validators per case | 8 | 20 |
| Repetitions / samples | 1 / 1 | 1 / 1 |
| Full input token upper bound, per invocation | 16,384 | 32,768 |
| Output token bound, per invocation | 2,048 | 4,096 |
| Provider response / text bytes | 1 MiB / 64 KiB | 1 MiB / 64 KiB |
| Root operation time, including queue | 180 seconds | 900 seconds |
| Queue wait from enqueue, excluding approval | 60 seconds | 300 seconds |
| Provider call deadline | 45 seconds | 60 seconds |
| Active operations per actor | 1 running, 1 queued/approval | 2 running, 3 queued/approval |
| Workspace active operations | n/a | 4 running, 10 queued/approval |
| Shared hosted pool | 20 running, 100 queued | 100 running, 1,000 queued/approval |
| Admission requests per actor per minute | 10 | 30 |
| SSE connections per actor | 2 | 2 |
| Session operations / messages / artifact revisions / requirements | 100 / 200 / 100 / 100 | same |
| Sessions per actor | 100 | 100 |

The general graph ceiling is 12/128 model calls, 4/6 helper calls, 6/12 tool calls and 1/2 pre-dispatch retries. **The actual V1 capabilities are stricter:** zero tools, zero automatic provider retries, at most two authoring calls (one initial plus one structured repair), and no evaluator repair. A check makes `cases × (one target + evaluators)` calls, at most 6/60. It never invokes helpers. Each operation executes one agent version; a comparison references one prior run rather than executing both versions again. No recursive coordinator loop exists.

Every provider call must fit the immutable graph's call count and remaining reserved maximum. The general $10 operation ceiling does not enlarge the $1 trial lifetime balance. Workspace operations above $0.25 require approval; smaller submissions run under the disclosed $0.25 automatic limit. Quotes expire after 24 hours. Every conversation has one active writer, including pending approvals.

The complete graph and maximum spend are computed before reservation. An oversized imported pack can be reviewed but cannot be silently reduced to three cases; the user must explicitly make a smaller draft or use the advanced builder.

## Model accounting and retries

The gateway counts the complete assembled invocation immediately before each call: system prompt, selected conversation history, structured requirements, artifacts, evidence, current message, response format and tool definitions (empty in V1). Each role is counted independently. Only the last six conversation messages and latest proposal are optional context; structured accepted/proposed/superseded requirements remain separate from that history. Observations are explicitly identified excerpts; full evidence remains stored.

`tiktoken-go/tokenizer` uses `o200k_base` for a diagnostic estimate. Admission uses the more conservative serialized UTF-8 byte bound plus a conformed framing allowance. This intentionally rejects some contexts that would fit a model's actual token window. It is not a claim that the latest user message is the full input or that every OpenRouter model shares a tokenizer. Unverified model families are unavailable. Observed input/output usage exceeding the bound disables the profile. Expired or missing profiles cannot authorize execution.

Provider route, model, role, price profile, token bounds, output limit, temperature, response format, request hash, generation ID, usage and actual output are persisted per attempt. OpenRouter routing uses only the configured `openai` provider, `allow_fallbacks:false`, required parameter support and maximum input/output price. There is no provider URL, credential reference or routing override accepted from the conversation.

Before network I/O the transaction creates a unique `(operation_id, step_key)` attempt in `DISPATCHING`. Temporal's paid activity has `MaximumAttempts=1`; the HTTP adapter has no application retry loop. A second dispatch of the same step fails. A provider success followed by worker death therefore leaves a durable uncertain attempt, not permission to charge again. Generation IDs and streamed text are journaled as they arrive. When no generation ID survived, the hold needs manual accounting review.

The reconciler makes bounded GET requests to OpenRouter's fixed generation endpoint, starting after 90 seconds and stopping automatic lookup after 24 hours. It accepts only the exact recorded generation ID and numeric cost. It never reruns inference or invents lost output. Reconciliation continues when `VIBE_ENABLED=false`, provided the key remains configured. Unknown holds do not expire by TTL. Operator resolution must be based on provider evidence, including verified zero cost where applicable.

## Execution and billing state

`CanTransition` is enforced within the same DB transaction that appends a state event:

```mermaid
stateDiagram-v2
    CREATED --> VALIDATING
    VALIDATING --> RESERVED
    VALIDATING --> AWAITING_APPROVAL
    AWAITING_APPROVAL --> VALIDATING: approve
    AWAITING_APPROVAL --> EXPIRED
    RESERVED --> QUEUED
    QUEUED --> RUNNING
    QUEUED --> EXPIRED
    QUEUED --> FAILED
    RUNNING --> FINALIZING
    FINALIZING --> COMPLETED
    FINALIZING --> PARTIAL
    FINALIZING --> FAILED
    RUNNING --> CANCELLING: Stop
    CANCELLING --> CANCELLED
```

Stop is also allowed before dispatch. `AWAITING_INPUT` is reserved in the enum; a conversational question currently completes its message operation and waits for a new message. Terminal operations cannot reenter running. Expiry rechecks state and deadline under the dispatch lock, so it cannot cancel a worker that already started. Approval starts the queue clock at enqueue, not the original quote creation time.

Billing separately follows `UNRESERVED → RESERVED → SETTLED / RELEASED / RECONCILING`. Stop ends future dispatch and signals local provider cancellation, but a request already sent may still finish and bill. The UI can therefore show **Execution: cancelled · Billing: reconciling**. The entire operation hold remains until every attempt is accounted. Known over-ceiling costs freeze funding/profile access for review. Refunds/disputes freeze the credit wallet and preserve verified event evidence; they do not rewrite provider usage or auto-release reservations.

All money uses PostgreSQL `bigint` / Go `int64` nano-USD. Available balance is `balance − held`. Grants, availability checks, reservations and settlement share a short DB transaction with row locks and a Vibe advisory lock. This deliberately serializes small V1 accounting critical sections; external network calls never occur inside them. Transactions time out after five seconds. Two $0.80 reservations against $1 cannot both succeed.

Credits deduplicate by payment ID, not webhook delivery ID. Included allowances deduplicate by subscription billing period. Top-up amount, currency, product, quantity and organization must match the durable checkout intent. Refund/dispute review markers arriving before payment success also freeze the eventual grant. Included credits do not currently expire. Automatic top-ups and automated dispute resolution are excluded.

## Safety boundaries and result meaning

- Requests are bounded before parsing; JSON rejects duplicate keys, malformed Unicode, excessive nesting/arrays/fields, and all `$ref` variants. YAML is a conservative subset: no anchors, aliases, merge keys, custom tags or multiple documents. A lexical/depth preflight precedes the unexpanded YAML node parser. This V1 uses bounded in-process parsing, **not** a separately memory-limited parser process.
- The supported deterministic set is contains, exact/normalized/fuzzy match, regex, numeric match, token F1 and bounded JSONPath checks. Resolved regexes are limited to 1 KiB; fuzzy operands are bounded. Arbitrary JSON Schema validators and recursive schemas are unavailable in this preview. The advanced runtime's existing JSON Schema resolver also has no remote loader by default.
- Test prompts remain evaluation data exactly, including adversarial strings. They are never tool arguments to a coordinator or role/system messages. The coordinator has no tools. Only authenticated typed HTTP actions can modify accepted requirements, run work or save resources. Expected-answer fields are withheld from target input where the pack declares them separately.
- Generated replies, evidence and uploads use `react-markdown` with raw HTML skipped, no images, no MDX, and restricted links. Nothing fetched from a pack/model becomes executable frontend code.
- There is no generic HTTP/URL inspection. Users paste text or import local JSON/YAML. The two external integrations are the server-configured model provider and existing verified billing integration. The advanced connector/agent execution paths remain outside this preview.
- A case passes only when all its checks pass. A known failed check makes a case FAIL even if another check is missing; check coverage and PARTIAL execution still disclose that gap. Rubrics require the top configured rating for PASS. This preview's binary counts are distinct from the advanced runner's weighted multidimensional scores, which remain in the saved pack.
- Permissions are rechecked in the DB at dispatch, before every paid call, before sensitive writes and on evidence reads. Archived accounts/organizations/workspaces and revoked memberships are denied. Resource IDs alone never authorize a read.
- Client message IDs are content-bound and unique per session. Edits use revision CAS. Reconnecting reads a consistent snapshot and never submits an operation. Full case evidence is fetched separately rather than retransmitted on each SSE tick. Both persisted events and the session revision guard against stale snapshots.

## Trial economics

The trial has $1 lifetime provider spend, 20 authoring submissions, 40 model calls, one initial check and one retest. All conversation/build/playground calls additionally consume a $0.25 exploration bucket and are limited to 28 calls. The remaining $0.75 and 12 calls are protected for the check/retest. Reaching the 20-message limit blocks further authoring, not the two checks. Admission and every paid dispatch enforce the call partition independently. Cookie rotation cannot bypass the independently reserved daily and campaign subsidy pools; exhausting either stops new subsidized execution. Workspace-funded execution and existing advanced BYOK flows remain independent of those pools.

`TestTrialEconomicsAcrossPilotModels` simulates 81,000 journeys across all nine assistant/target choices, with the evaluator fixed to GPT-4.1 Mini. Inputs are **synthetic scenario assumptions**, not measured production P50/P90 or live price discovery. At the scenario prices, the default Mini/Mini first check is approximately $0.0388 P50 / $0.0519 P90; the complete journey is $0.0739 / $0.0988. The most expensive exposed pair is approximately $0.1549 / $0.2187 for first check and $0.2914 / $0.4140 for the full scenario. Long expensive conversations may hit the exploration bucket earlier; the message count is a ceiling, not a promise of twenty expensive responses. Check/retest funds remain protected.

Launch still requires a representative live conformance/latency cohort and updated prices for every exposed route. Development verification uses fake providers and makes no paid calls. Operator-approved profiles remain the launch gate.

## Dependencies

| Need | Choice | New dependency? |
| --- | --- | --- |
| Model/provider transport | Existing `runtime/provider` OpenAI-compatible adapter and `net/http` | No; adds explicit bounded options and generation/cost metadata |
| Token estimate | `github.com/tiktoken-go/tokenizer v0.7.0`, `o200k_base` | Yes; existing code had no suitable tokenizer. v0.8.1 requires Go 1.26, so v0.7.0 preserves Go 1.25.5 |
| Structured output/import validation | `encoding/json`, strict Go structs, bounded walker, existing pack compiler | No |
| YAML | Existing `gopkg.in/yaml.v3 v3.0.1` node parser | No; promoted to direct backend dependency |
| JSON Schema in advanced runtime | Existing `github.com/google/jsonschema-go v0.3.0` | No; Vibe does not expose arbitrary schema validators |
| Distributed limiting | Existing `github.com/redis/go-redis/v9 v9.18.0`, bounded Lua scripts | No |
| Markdown | `react-markdown 10.1.0` + existing `remark-gfm` | Yes; safe untrusted rendering without `next-mdx-remote` execution |
| Streaming | Existing HTTP server + native browser fetch/SSE | No |
| Transactions | Existing `github.com/jackc/pgx/v5 v5.8.0` | No |
| Durable execution | Existing Temporal SDK v1.41.0 | No |
| Browser tests | `@playwright/test 1.63.0` | Yes; reproducible reconnect/cancel/duplicate-submit journeys |
| Unit/integration tests | Existing Go tests, Temporal testsuite, miniredis, Vitest; real isolated PostgreSQL | No |
| Metrics/tracing hooks | Existing OpenTelemetry and Prometheus stack | No; counters export with `METRICS_ENABLED`. Trace spans use the existing OTel API; a trace exporter is not added |

No LangChain, LangGraph, vector database, model SDK framework or new message broker is added. Both npm and pnpm lockfiles are updated because frontend CI uses npm.

## Configuration and local verification

### Free-model localhost pilot

`VIBE_FREE_ONLY=true` permits only explicit conformed profiles with `free:true`, zero input/output prices, and a supported exact model/provider pair. The current pairs are `dots-studio/dots-3-note-preview:free` / `atlas-cloud/fp8`, `liquid/lfm-2.5-2.6b:free` / `liquid/fp8`, and `google/gemma-4-31b-it:free` / `google-ai-studio`. `VIBE_DEFAULT_MODEL` sets all three initial roles and the trial evaluator. Paid models are rejected in this mode. Free models do not use an o200k estimate; each invocation still uses the conservative UTF-8 byte bound plus its verified framing allowance.

The provider request explicitly caps prompt, completion and per-request prices at zero, pins the endpoint and disables fallbacks. A profile may set `disable_reasoning:true` only for a route that supports disabling reasoning. This setting is persisted with the attempt. Mandatory-reasoning models may be too slow or exhaust the output budget for draft generation.

Migration **00073** permits explicit zero-price attempts and reservations. Existing call/time/concurrency limits and durable accounting remain in force, including unknown outcomes. The installation additionally allows at most **40 free attempts per UTC day**; failures also count. Provider-side quotas may be lower. A zero price is never inferred from missing usage or configuration.

The September 7 local pilot uses Dots 3 Note with reasoning disabled. Its conformance probe and a real browser draft/check journey returned numeric cost `0`. Liquid's small probe also succeeded, but full draft generation timed out, and Gemma was rate-limited upstream. The timeout remains an uncertain attempt; it was not automatically retried or recorded as verified zero cost. These observations are not a production reliability claim.

The first live browser pass exposed an empty-session serialization bug (a null operation list) and poor generated checks that required mutually exclusive phrases on every case. New sessions now serialize an empty array. Vibe's authoring instructions explicitly describe global check scope and recommend a nonempty-output validator plus a conditional semantic assertion for conversational policies. Existing results and accepted criteria remain unchanged; generated evaluation quality still needs human review.

Verification after the fixes: six successful Dots calls recorded total provider cost `0`, including the initial draft/check and a new draft with conditional semantic criteria. The new draft was not evaluated again under the already-used anonymous initial-check allowance. Five mocked Playwright journeys passed, along with the PostgreSQL race tests, provider/scoring tests, backend build/vet, TypeScript and focused lint. The live draft screenshot was inspected. The earlier Liquid timeout remains `UNCERTAIN`.

Credentials belong in ignored `backend/.env` with mode `0600`. The pilot runs in isolated local PostgreSQL (`vibe_local`), Redis and Temporal. The browser is at `http://localhost:3000/vibe-evals`, API at `http://localhost:55440`, and Temporal UI at `http://localhost:58233`. WorkOS placeholder settings permit the anonymous preview; real sign-in and saving require the application's WorkOS development configuration.

### Hosted paid pilot

Apply migrations 00070–00073 through the normal deployment migration process. API and worker need the same PostgreSQL, Redis and Temporal configuration. The feature remains closed to hosted calls without explicit configuration:

```dotenv
VIBE_ENABLED=false
NEXT_PUBLIC_VIBE_EVALS_ENABLED=false
VIBE_COOKIE_SECRET=<at-least-32-random-bytes>
VIBE_OPENROUTER_KEY=<server-only-key>
VIBE_CAMPAIGN=<stable-pilot-identifier>
VIBE_ANON_DAILY_USD=50
VIBE_ANON_CAMPAIGN_USD=500
REDIS_URL=redis://127.0.0.1:6379
TEMPORAL_HOST_PORT=127.0.0.1:7233
FRONTEND_URL=http://localhost:3000
CORS_ALLOWED_ORIGINS=http://localhost:3000
```

`VIBE_MODELS_JSON` contains a bounded array of profiles. Example shape below is deliberately **not conformed**, so it cannot authorize a call. Replace scenario prices/expiry with verified provider ceilings; add a profile for each exposed model and set `conformed:true` only after route, context, usage accounting and output-limit verification.

```json
[{"id":"openai/gpt-4.1-mini","name":"GPT-4.1 Mini","route":"openai","input_nano_per_token":400,"output_nano_per_token":1600,"context_tokens":128000,"framing_allowance":2048,"conformed":false,"expires_at":"2026-09-20T00:00:00Z"}]
```

Missing/expired pricing, Redis failure, wallet failure or an unsafe cost bound rejects new hosted execution. Missing/zero anonymous subsidy configuration disables only subsidy admission. Pool amounts are immutable grants for a given campaign/day identity; do not change campaign IDs to bypass a spent ceiling.

Optional `VIBE_TOPUP_PRODUCTS_JSON` entries are `{id, credits_nano_usd, price_minor, currency:"USD"}` mapped to actual Dodo products. `VIBE_PLAN_ALLOWANCES_JSON` maps Dodo subscription product IDs to nano-USD allowances. These require the existing verified Dodo webhook configuration; no fake/dev endpoint can grant public credits.

Local commands (no production credentials):

```sh
# From backend/, with an isolated migrated database whose name contains vibe_test:
VIBE_TEST_DATABASE_URL='postgres://agentclash:agentclash@127.0.0.1:55439/vibe_test?sslmode=disable' \
  go test -race ./internal/vibe ./internal/api -run 'Test(Vibe|Integration|Structured|YAML|Fanout|Money|FullContext|Unknown|Coverage|Requirement|StateMachine|MissingPricing|TrialEconomics|PaidTemporal|InvalidJudge)' -count=1
go build ./...
go vet ./...

# From runtime/:
go test ./provider ./scoring -count=1

# From web/, run browser and type generation sequentially:
pnpm exec playwright install chromium
pnpm exec playwright test --config playwright.vibe.config.ts
pnpm exec tsc --noEmit --incremental false
pnpm exec eslint src/app/vibe-evals src/components/vibe src/lib/vibe.ts
pnpm exec vitest run src/components/vibe/safe-markdown.test.tsx
```

The browser config starts a localhost Next server and mocks the API. Optional `VIBE_TEST_CHROMIUM` selects an existing Chromium executable. It needs no AI/Dodo keys. `.github/workflows/vibe-evals.yml` runs the database failure tests and mocked browser journeys in CI. Local verification is recorded here; remote CI status must be checked separately.

For an actual local app, use the repository's normal local stack, `go run ./cmd/api-server`, `go run ./cmd/worker`, and `pnpm dev` with local DB/Redis/Temporal and WorkOS development setup. Enable both feature flags only after provider configuration. Example private session creation:

```sh
curl -c /tmp/vibe-cookie -H 'Content-Type: application/json' \
  -d '{"id":"a178b4de-3ee8-414c-9f5e-e7ea45409bd3"}' \
  http://localhost:8080/v1/vibe/sessions
# Reading this ID requires that capability cookie, or its claimed account:
curl -b /tmp/vibe-cookie \
  http://localhost:8080/v1/vibe/sessions/a178b4de-3ee8-414c-9f5e-e7ea45409bd3
# Only after enabling verified profiles and local funding configuration:
curl -b /tmp/vibe-cookie -H 'Content-Type: application/json' \
  -d '{"client_id":"597609ce-00b6-49d8-b0a2-1a35d5fb3dd9","revision":0,"kind":"message","content":"Build a support agent. Refunds are allowed within 30 days; escalate unclear cases.","models":{"assistant":"openai/gpt-4.1-mini","target":"openai/gpt-4.1-mini","evaluator":"openai/gpt-4.1-mini"}}' \
  http://localhost:8080/v1/vibe/sessions/a178b4de-3ee8-414c-9f5e-e7ea45409bd3/messages
```

The last command uses hosted inference when enabled and is not part of the no-charge development checks. Update the session revision for subsequent submissions; reconnect using GET/SSE rather than resending with a fresh ID.

## Review checklist mapped to the 24 concerns

| Concern | Implementation/evidence |
| --- | --- |
| 1 request/expansion sizes | Limits, HTTP MaxBytesReader, bounded parser, compiled case/operand caps, artifact/document/operation ceilings; oversized/chunked tests |
| 2 complete context | Per-invocation `CountContext`; full system/tools/history test; role-specific journal snapshots |
| 3 autonomous loops | Fixed graphs, durable call counters, zero tools/retries, single authoring repair, root deadline and dollar hold |
| 4 duplicate provider spend | Pre-I/O journal uniqueness, paid Temporal activity one attempt, fixed-ID reconciliation; crash/duplicate tests |
| 5 Stop vs billing | Independent state, cancellation polling, retained uncertainty; mid-call Stop integration and browser tests |
| 6 wallet concurrency | Atomic reservation and balance row locks; simultaneous $0.80 approvals against $1; payment-ID dedupe tests |
| 7 fail closed | Profile, Redis and wallet checks at admission/dispatch; nil/offline Redis and missing pricing tests |
| 8 fan-out | Complete graph/cost before reservation, no implicit sampling; 3/20 cases, 1/2 judges, one version executed; quote above automatic threshold |
| 9 pack fallback | Strict compiler rejects reduced coverage; compiles and executes composed result; generation/promotion regression tests |
| 10 model independence | Role validation at durable attempt creation plus wire policy tests and a three-distinct-model service journey |
| 11 technical unknowns | Separate verdict, check error and execution state; invalid judge and timeout tests, including stopped fan-out |
| 12 scorecard arithmetic | `Aggregate` over persisted result metadata; known FAIL plus missing evaluator coverage test |
| 13 injection as data | Adversarial strings preserved in compiler tests; no model tools; accepted blueprint cannot be weakened in coaching |
| 14 URL fetching | No user-selectable HTTP tool/importer; unsupported capabilities rejected |
| 15 structured imports | No refs/aliases, encoded size/nesting checks, strict compiler and local export round-trip test |
| 16 Markdown | Safe renderer and browser/Vitest HTML/image/script tests |
| 17 memory provenance | Structured proposed/accepted/rejected/superseded requirements, source/proposal message IDs, explicit acceptance/replacement tests |
| 18 tabs/reconnect | IDs, CAS, DB snapshots/events; duplicate and stale revision tests; four browser journeys include old snapshots and refresh |
| 19 execution authorization | DB checks at dispatch/calls/write/evidence; revoked membership and private case-access tests |
| 20 aggregate subsidy | Daily/campaign account reservations in the same transaction; exhaustion rollback test; no config means no subsidy |
| 21 economics | 81,000 synthetic journeys, protected dollars/calls, real DB test of 20 messages plus playground followed by both checks; real cohort/conformance still a launch task |
| 22 libraries | Exact choices above; no orchestration framework added |
| 23 state machines | Go transition guard, SQL enums, persisted events, separate billing states and expiry race test |
| 24 failure testing | Real PostgreSQL, fake providers/Redis outages, Temporal testsuite, shared engine regressions, browser failures and new CI job |

## Ongoing monitoring and intentional V1 exclusions

The first scorecard starts the relationship by preserving the agent instructions, evaluation revision, model policies and evidence. Saving moves useful work into the existing builder and agent registry. That is the handoff to larger evaluations, connected deployments, datasets, baselines, regression suites and release gates.

V1 does not pretend that this handoff enables monitoring. Continuous trace ingestion setup, sampling policy, alerts, production budgets, scheduled comparisons and automatic failure-to-regression promotion need their own explicit workspace configuration. Existing trace/dataset/baseline/regression/gate systems remain available; they are not silently activated by a preview.

Also excluded: arbitrary external agents/URLs/code execution in Vibe, remote connectors, file archives/PDF/image understanding, recursive schemas, simultaneous multi-version runs, repetitions/consensus, automatic publishing/deployment, a full history search UI and BYOK inside this conversation. Existing advanced BYOK workflows remain available outside Vibe.

## Accounting recovery runbook

1. Inspect `vibe_operations` with `billing='RECONCILING'`; join `vibe_attempts` by operation ID and find missing `actual_cost`/generation IDs. Do not log or export prompts/keys in metrics.
2. Generation lookup may settle cost automatically for 24 hours. If output was lost, cases remain UNKNOWN even after billing settles.
3. For unmatched/aged attempts, obtain verified provider evidence. The trusted `Store.ReconcileCost` function performs the accounting-only update transaction. There is no public reconciliation or grant endpoint.
4. Never release uncertain holds by age or resend the original request. Over-ceiling usage and refunds/disputes require operator review of the relevant profile/account before re-enabling it.
5. Observe `vibe.model.calls`, `vibe.model.uncertain_calls`, `vibe.model.cost_nano_usd` and durable DB backlog/holds. Existing OTel metrics export through the configured Prometheus endpoint. A dedicated alerting dashboard/exported trace backend is not included in this change.

Provider contracts consulted: [OpenRouter routing](https://openrouter.ai/docs/guides/routing/provider-selection), [usage accounting](https://openrouter.ai/docs/cookbook/administration/usage-accounting), [Dodo payment payload](https://docs.dodopayments.com/developer-resources/webhooks/intents/payment), [refund payload](https://docs.dodopayments.com/developer-resources/webhooks/intents/refund), [dispute payload](https://docs.dodopayments.com/developer-resources/webhooks/intents/dispute). Initial implementation verification used fake providers. The subsequent user-authorized free-only localhost pilot is documented above; no purchase or public deployment was performed.
