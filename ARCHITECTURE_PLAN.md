# AgentClash Architecture Plan

Status: canonical architecture and tech stack plan

Companion document: `PRODUCT_STRATEGY.md`

Last updated: 2026-03-12

## 1. Purpose of this document

This document turns the product strategy into an implementation-ready architecture plan for the app, the backend, the data model, the infrastructure, and the developer workflow.

It is intentionally opinionated. The goal is not to list possible architectures. The goal is to choose one that is strong enough for a serious v1 product, but simple enough for a small team to ship quickly.

This plan assumes the product defined in `PRODUCT_STRATEGY.md`:

- cloud-first SaaS
- B2B private-workspace revenue core
- public arena as growth loop
- engineering-wide benchmark surface

## 2. Current repo and what should be preserved

The current repo is not the product, but it contains the right kernel.

Keep and evolve these parts:

- the Go execution engine
- provider abstractions
- challenge packaging model
- telemetry and scoring concepts
- live event/replay mindset

Do not preserve the current packaging as the long-term product shape:

- `cmd/web` should not become the production app shell
- `cmd/race` should not remain the primary way races are orchestrated
- `race.yaml` should not remain the main product control surface
- local filesystem result storage should not remain the source of truth
- in-memory events should not remain the realtime layer

### Keep vs replace

Keep:

- `internal/engine` as the executor kernel
- `internal/provider` as the starting point for the provider layer
- `internal/scoring` as the starting point for scorecards
- `internal/telemetry` as the seed of the replay/event model

Replace or split:

- `cmd/web` with a real web app plus API server
- `cmd/race` with a worker-driven orchestration flow
- file-based config with database-backed objects
- in-memory event fanout with durable streams

## 3. Architecture principles

These principles should guide every architectural decision.

### Principle 1: keep the product monorepo

AgentClash is still early. The team should optimize for speed, shared types, and coordinated changes rather than repo sprawl.

### Principle 2: separate control plane from execution plane

The product needs a strong boundary between:

- app and API logic
- orchestration
- sandboxed run execution

This is the most important architectural split.

### Principle 3: choose managed infrastructure where it saves product time

Do not spend the first year building auth, billing, workflow orchestration, or sandbox infrastructure that can be bought. Spend the first year building:

- the benchmark product
- replay UX
- scorecards
- challenge packs
- public and private product loops

### Principle 4: make replay and benchmarking first-class data products

Runs are not just jobs. They create historical product assets:

- replays
- scorecards
- leaderboard data
- category-level insights

The data model should reflect that from day one.

### Principle 5: phase complexity

V1 should not start with a dozen microservices. It should start with a small number of well-defined services and clear upgrade paths.

## 4. Recommended target architecture

The recommended v1 architecture is:

```text
Browser / Public Arena / Private Workspace
                |
                v
        Next.js Web App
                |
                v
         API Server (Go)
                |
      -----------------------
      |          |          |
      v          v          v
 PostgreSQL    Redis     Temporal
      |          |          |
      |          |          v
      |          |      Worker Service (Go)
      |          |          |
      |          |          v
      |          |    Sandbox Provider
      |          |     (E2B in v1)
      |          |
      v          v
Object Storage  Realtime Fanout
   (S3)        (Redis pub/sub)
```

### What each layer does

#### Web app

Owns:

- public arena UI
- private workspace UI
- auth entry points
- replay UI
- leaderboard UI
- billing pages

#### API server

Owns:

- product API
- org/workspace/challenge/agent/run CRUD
- authz checks
- run submission
- run state materialization
- webhooks
- realtime session registration

#### Temporal

Owns:

- workflow orchestration
- retries
- timers
- cancellations
- child workflows for multi-agent runs
- durable step progression

#### Worker service

Owns:

- preparing workspaces
- calling provider layer
- running tool loops
- streaming events
- persisting artifacts
- finalizing scorecards

#### Sandbox provider

Owns:

- isolated execution environment for file operations and command execution

In v1 this should be externalized via E2B. In v2+, AgentClash can decide whether it wants to keep a vendor-backed model or move high-volume workloads to self-managed Firecracker-based workers.

## 5. Tech stack decisions

This section is intentionally explicit. These are the recommended choices.

### Frontend

#### Web framework

- Next.js 15
- React 19
- TypeScript 5

Why:

- best balance of public site, app shell, SSR, and SEO
- supports public arena pages and authenticated workspace UI in one product
- strong developer ecosystem

#### Styling

- Tailwind CSS 4
- local component library built from Radix primitives where needed

Why:

- fast iteration
- enough control for a distinctive arena-style interface
- keeps design system local to the repo

#### Client data and live state

- TanStack Query for API data fetching and caching
- native WebSocket client for live run events
- Zustand only for local transient replay/live state where React state becomes awkward

Why:

- predictable server-state handling
- minimal state-management overhead

#### Frontend error reporting

- Sentry for browser/runtime error monitoring

Why:

- fast setup
- good for a small team

### Backend

#### Language and runtime

- Go 1.25+

Why:

- existing repo already uses Go
- current engine is already in Go
- good fit for concurrency, workers, and low-overhead APIs

#### API framework

- `chi` for HTTP routing
- plain JSON REST for public and app APIs
- WebSocket endpoint for live run streaming

Why:

- simplest surface for the current stage
- avoids premature complexity from gRPC-first designs
- easy for frontend and partner integrations

Important decision:

- do not introduce gRPC/Connect in v1
- only add internal RPC contracts later if the provider gateway or scoring service becomes a separate service

#### Data access

- `pgx` for PostgreSQL driver
- `sqlc` for typed query generation
- SQL migrations with `goose`

Why:

- typed query generation reduces ORM drift
- simple, explicit SQL is better for benchmark/reporting products
- easy to review and debug

#### Background workflows

- Temporal Cloud

Why:

- run orchestration is durable and long-running
- retries, timers, cancellation, and child workflows matter a lot here
- better fit than hand-rolled background jobs once multi-agent run state becomes product-critical

Important decision:

- do not build workflow orchestration on ad hoc goroutines plus Redis queues
- do not add Kafka just to solve workflow durability

### Data layer

#### Primary relational database

- PostgreSQL 17

Owns:

- orgs
- users
- workspaces
- challenge metadata
- agent builds
- run metadata
- scorecards
- leaderboard materializations
- billing metadata

#### Cache and ephemeral state

- Redis 7

Owns:

- rate limits
- short-lived caches
- pub/sub for realtime fanout
- presence and live viewers
- transient run tail buffers

#### Artifact storage

- AWS S3 in production
- MinIO locally if needed

Owns:

- raw traces
- replay blobs
- logs
- challenge assets
- exported reports
- run attachments

#### Analytics store

- v1: PostgreSQL plus S3 only
- v2: ClickHouse Cloud for high-volume replay analytics and leaderboard/report queries

Important decision:

- do not start with ClickHouse in the first sprint
- design event schemas so ClickHouse can be added later without rewriting the product model

### Authentication and organization management

- WorkOS for auth, org membership, and SSO path

Why:

- strong B2B fit
- faster time-to-market than building org auth from scratch
- enterprise path without early auth complexity

Important decision:

- auth is outsourced
- authorization remains in AgentClash

### Billing

- Stripe

Why:

- standard SaaS billing stack
- good enough for subscriptions, metered usage, and enterprise invoicing later

### Observability

- OpenTelemetry for traces, metrics, and logs
- OTEL Collector in backend environments
- Grafana Cloud as the initial observability backend
- Sentry for app errors

Why:

- avoids instrumentation lock-in
- works across Go workers and Next.js frontend
- managed backend saves operator time

### Sandbox execution

- v1 sandbox provider: E2B
- abstraction layer in code: `SandboxProvider`
- v2 option: Firecracker-based self-managed runner if costs justify it

Why:

- public and private engineering challenges need safer execution than local host shelling
- buying sandbox capability early is better than building microVM orchestration too soon

### Infrastructure and deployment

#### Frontend hosting

- Vercel for the Next.js app

Why:

- best fit for public pages plus authenticated app routes
- very fast iteration loop

#### Backend hosting

- AWS ECS/Fargate for API server and workers initially
- RDS for PostgreSQL
- ElastiCache for Redis
- S3 for object storage

Why:

- managed enough for a startup
- avoids Kubernetes too early
- clean path to autoscaling workers

Important decision:

- do not start on Kubernetes
- only revisit K8s when dedicated sandbox worker fleets or custom runtime scheduling require it

## 6. Recommended repo layout

Keep a monorepo and evolve it like this:

```text
/apps
  /web                  # Next.js app
/cmd
  /api-server           # Go control-plane API
  /worker               # Go worker entrypoint
/internal
  /api                  # HTTP handlers, authz, DTOs
  /core                 # domain logic
  /engine               # preserved execution kernel
  /provider             # provider integrations
  /sandbox              # sandbox abstraction and E2B adapter
  /scoring              # scorecards and ranking
  /telemetry            # event and replay model
  /workflow             # Temporal workflows and activities
  /store                # sqlc-generated queries and repositories
/migrations
/infra
  /terraform
/docs
  optional later
```

### What to do with current paths

- keep `internal/engine`, but stop treating it as the whole app
- replace `cmd/web` over time with `apps/web` plus `cmd/api-server`
- replace `cmd/race` with `cmd/worker`
- move static web assets out of the embedded-Go UI pattern

## 7. Service boundaries

V1 should start with only three code-bearing runtime services.

### 1. Web app

Responsibilities:

- render public pages
- render authenticated app UI
- call API server
- open live event stream via WebSocket

Should not do:

- direct database access
- workflow orchestration
- provider calls

### 2. API server

Responsibilities:

- authenticate users
- authorize org/workspace access
- own CRUD APIs
- enqueue or start Temporal workflows
- serve replay and leaderboard data
- manage billing hooks and integrations

Should not do:

- run agent loops
- execute shell commands
- manage sandbox lifecycles directly

### 3. Worker

Responsibilities:

- execute run workflows and activities
- interact with sandbox provider
- call providers
- stream events
- store artifacts
- finalize scorecards

Should not do:

- own primary product CRUD
- own auth
- serve end-user HTTP traffic

### Services to defer until later

Do not split these out in v1:

- dedicated provider gateway
- dedicated realtime gateway
- dedicated scoring service
- dedicated analytics ingester

These can remain modules inside the API server or worker until scale justifies separation.

## 8. Core data model

This is the recommended logical schema for v1.

### Identity and orgs

- `users`
- `organizations`
- `organization_memberships`
- `workspaces`
- `workspace_memberships`

### Product catalog

- `challenge_packs`
- `challenge_pack_versions`
- `challenge_tasks`
- `challenge_assets`
- `scorecard_definitions`

### Agent and provider config

- `agent_builds`
- `agent_build_versions`
- `provider_accounts`
- `provider_credentials_ref`
- `tool_policies`

### Run system

- `runs`
- `run_agents`
- `run_steps`
- `run_artifacts`
- `run_events_index`
- `run_scorecards`
- `run_exports`

### Ranking and reporting

- `leaderboards`
- `leaderboard_entries`
- `benchmark_reports`

### Commercial

- `billing_accounts`
- `billing_subscriptions`
- `usage_counters`

### Important data decisions

- store run state and scorecard summaries in PostgreSQL
- store large replay payloads and full artifacts in S3
- keep a lightweight index in PostgreSQL that points to S3 objects
- make every run and artifact addressable by stable IDs

## 9. API shape

Use JSON REST externally in v1.

### Main resource groups

- `/v1/me`
- `/v1/organizations`
- `/v1/workspaces`
- `/v1/challenge-packs`
- `/v1/agent-builds`
- `/v1/runs`
- `/v1/replays`
- `/v1/leaderboards`
- `/v1/billing`
- `/v1/webhooks`

### Core write flows

#### Create an agent build

User creates or versions an agent build with:

- model
- provider account
- prompt strategy reference
- tool policy
- timeout profile

#### Start a run

User submits:

- workspace
- challenge pack version
- one or more agent builds
- optional scorecard override
- visibility mode

API server:

- validates permissions
- creates `run`
- starts Temporal workflow

#### Stream live run state

Frontend connects to:

- `/v1/runs/:id/live`

Use:

- WebSocket for event stream
- fallback to polling for non-live or degraded cases

#### Fetch replay

Frontend requests:

- run summary from PostgreSQL
- replay steps and artifacts through indexed S3-backed payloads

## 10. Realtime architecture

V1 realtime should stay simple and robust.

### Recommended path

- workers publish run events to Redis pub/sub
- API server subscribes and fans out to WebSocket clients
- important run state transitions are also persisted to PostgreSQL
- full replay payloads and attachments are stored in S3

### Why this is the right v1 decision

- enough for live arena and workspace run views
- easy to reason about
- avoids premature event-platform complexity

### What not to do in v1

- do not introduce Kafka
- do not create a separate realtime service
- do not rely only on pub/sub without persisted run state

## 11. Workflow design

Temporal should model the lifecycle of a run explicitly.

### Parent workflow: RunWorkflow

Responsibilities:

- initialize run
- fan out child workflows for each agent build
- track deadline and cancellation
- wait for children to complete
- invoke final scoring
- mark run complete

### Child workflow: AgentRunWorkflow

Responsibilities:

- provision sandbox
- prepare challenge workspace
- execute think/act loop
- emit events
- capture artifacts
- record failure or completion

### Activity groups

- `PrepareWorkspace`
- `CreateSandbox`
- `CallProvider`
- `ExecuteToolCall`
- `PersistEvent`
- `UploadArtifact`
- `FinalizeTrace`
- `ScoreRun`
- `MaterializeLeaderboardEntry`

### Failure handling rules

- provider failure should fail only the affected agent run unless the whole run policy says otherwise
- scoring failure should not erase run completion; it should mark the run as completed with scorecard pending or errored
- artifact upload failure should retry with backoff, then degrade gracefully with missing-artifact markers

## 12. Execution and sandbox design

This is the highest-risk technical area. The plan needs clear defaults.

### V1 execution model

- each agent run gets its own sandbox
- sandbox receives only the challenge workspace and allowed tools
- all filesystem writes happen inside the sandbox
- shell access is mediated by the sandbox provider, not the API host

### Tool policy model

Every challenge pack must declare:

- allowed tools
- shell allowance or denial
- network allowance or denial
- maximum workspace size
- maximum execution duration

### Sandbox provider abstraction

Create a `SandboxProvider` interface with operations like:

- create
- upload workspace
- exec command
- read file
- write file
- list files
- destroy

Do not let the engine depend directly on E2B types. The provider must be replaceable.

### Migration path

- v1: E2B for speed and safety
- v2: evaluate mixed mode
  - managed sandbox for public/untrusted runs
  - self-managed runner for high-volume private runs
- v3: Firecracker-based fleet only if economics justify it

## 13. Provider architecture

The current provider layer is a good start, but it must become a product-grade module.

### V1 provider responsibilities

- normalize model invocation
- capture token usage and cost metadata
- standardize retry categories
- standardize provider error classes
- enforce per-provider concurrency and timeout policies

### Provider account model

Support both:

- platform-managed provider accounts
- bring-your-own-key provider accounts

Important decision:

- v1 should support BYO provider credentials for serious teams
- managed billing through AgentClash can come later

### Provider module boundaries

Keep provider adapters inside the worker codebase in v1. Do not split a separate provider gateway service until:

- you need cross-product provider governance
- provider traffic becomes operationally distinct
- pricing/routing logic grows beyond a library concern

## 14. Scoring architecture

Scoring must stop being a single static formula.

### V1 scoring model

Each challenge pack version has a scorecard definition with:

- required metrics
- optional metrics
- weights
- pass/fail validators
- leaderboard eligibility rules

### Scorecard categories

- correctness
- completion
- speed
- cost
- reliability
- challenge-specific rules

### Ranking rules

Keep ranking materialized, not computed on every page request.

Store:

- canonical scorecard
- leaderboard-eligible score
- ranking breakdown

## 15. Replay architecture

Replay is not a debug log. It is a product surface. Treat it as such.

### Data split

Store in PostgreSQL:

- run summary
- step metadata
- pointers to large payloads

Store in S3:

- large tool outputs
- raw model payloads
- challenge attachments
- export bundles

### Replay model

Each replay step should be typed:

- planning step
- tool invocation step
- observation step
- judge/scoring step
- system event

### Replay UX requirement

The backend must support:

- step pagination
- artifact previews
- side-by-side replay comparisons
- summary rollups

That means the event schema must be stable and versioned.

## 16. Public leaderboard architecture

Leaderboards should be materialized views backed by explicit rules, not ad hoc queries.

### V1 model

Materialize leaderboard entries whenever:

- a run becomes complete
- a challenge pack version changes visibility
- a scorecard is recalculated

### Required leaderboard dimensions

- category
- challenge pack
- challenge pack version
- season
- public/private visibility
- org or global scope

### Ranking credibility requirements

Every leaderboard page must be able to answer:

- what tasks are included
- what rules were used
- what date range applies
- whether this is public, private, or curated

## 17. Security and compliance baseline

The product does not need enterprise certification on day one, but it does need a clean baseline.

### V1 security baseline

- WorkOS-backed auth
- encrypted provider credential storage using cloud KMS
- org/workspace-level authorization
- audit log for admin and billing actions
- sandboxed execution for code-capable challenges
- private-by-default workspace data
- signed artifact URLs for downloads

### Secrets handling

- do not store provider keys directly in application tables
- store references to secret manager entries or encrypted blobs
- rotate platform-managed credentials

### Network policy

- sandbox outbound network disabled by default
- only enable per challenge pack when explicitly required

## 18. Deployment topology

### Local development

- Next.js app running locally
- API server running locally
- worker running locally
- PostgreSQL via Docker Compose
- Redis via Docker Compose
- Temporal dev server locally or Temporal Cloud dev namespace

### Staging

- separate cloud environment
- smaller RDS/Redis instances
- real WorkOS and Stripe test mode
- limited sandbox budget

### Production

- Vercel for web
- AWS ECS/Fargate services:
  - api-server
  - worker
- RDS Postgres
- ElastiCache Redis
- S3
- Temporal Cloud
- E2B sandbox provider
- Grafana Cloud
- Sentry

### Why this topology is correct for v1

- managed enough to keep ops load low
- strong enough for a real product
- easy to scale the worker tier independently

## 19. CI/CD and developer workflow

### Monorepo workflow

- keep one repo
- root `Makefile` remains the cross-language entrypoint
- add `pnpm` workspace config once the Next.js app lands

### Backend CI

- unit tests
- integration tests against Postgres/Redis
- migration checks
- `go test ./...`

### Frontend CI

- typecheck
- lint
- build
- component or e2e smoke tests

### Release flow

- web app deploys independently
- API and worker deploy independently
- migrations run before API rollout

## 20. Testing strategy

The architecture should be validated at multiple levels.

### Unit tests

- scorecard logic
- provider adapters
- tool policy enforcement
- repository queries

### Integration tests

- API against Postgres and Redis
- Temporal workflow tests
- artifact upload/download path
- authz and workspace boundaries

### Sandbox integration tests

- workspace provisioning
- command execution
- file read/write
- network blocked by default

### End-to-end tests

- sign in
- create workspace
- create agent build
- start run
- watch live progress
- open replay
- inspect leaderboard

### Production validation

- canary runs on official challenge packs
- synthetic public arena run every deploy window

## 21. Migration plan from the current codebase

This is the recommended implementation order.

### Phase 1: preserve the kernel, add the app shell

- create `cmd/api-server`
- create `cmd/worker`
- create `apps/web`
- keep `internal/engine` largely intact
- stop investing in `cmd/web`

### Phase 2: database-backed product model

- add PostgreSQL schema
- replace `race.yaml`-driven run definition with DB objects
- replace filesystem-only run metadata with DB records

### Phase 3: workflow and realtime

- introduce Temporal workflows
- add Redis pub/sub live event fanout
- add replay and artifact storage in S3

### Phase 4: sandbox hardening

- replace local-host execution assumptions with sandbox provider abstraction
- integrate E2B
- challenge pack tool/network policy enforcement

### Phase 5: public arena and workspace polish

- build leaderboard and replay pages
- add billing and org flows
- add challenge pack management

## 22. Decisions explicitly rejected

These paths are rejected for now.

### Rejected: keep the Go embedded UI as the product frontend

Reason:

- not viable for the public/private product split
- weak for modern app UX

### Rejected: Kubernetes from day one

Reason:

- too much ops cost for current stage
- ECS/Fargate is enough for the initial service shape

### Rejected: microservices for every domain

Reason:

- too much coordination overhead
- modules inside a small number of services are enough initially

### Rejected: build our own workflow engine

Reason:

- orchestration durability is core and too easy to get wrong
- Temporal solves the right class of problems

### Rejected: self-managed microVM fleet in v1

Reason:

- operationally expensive
- delays product learning

## 23. Final recommendation

The correct architecture for AgentClash is:

- Next.js frontend
- Go API server
- Go worker
- PostgreSQL as the source of truth
- Redis for cache and live fanout
- Temporal for run orchestration
- S3 for artifacts and replay payloads
- WorkOS for auth
- Stripe for billing
- OpenTelemetry for instrumentation
- E2B-backed sandboxing in v1
- AWS for backend infrastructure
- Vercel for frontend hosting

This stack is not the most minimal possible stack, but it is the right one for the product. It keeps the current Go engine, buys the boring but difficult infrastructure where that saves time, and leaves clear upgrade paths when scale or cost requires bringing more of the platform in-house.

## 24. Official references consulted

- Next.js home and docs: https://nextjs.org/
- Go home: https://go.dev/
- PostgreSQL 17 docs: https://www.postgresql.org/docs/17/index.html
- Redis docs: https://redis.io/docs/latest/
- Temporal home: https://temporal.io/
- Temporal docs: https://docs.temporal.io/
- OpenTelemetry home: https://opentelemetry.io/
- ClickHouse home: https://clickhouse.com/clickhouse
- E2B home: https://e2b.dev/
- E2B docs: https://e2b.dev/docs
- Daytona home: https://www.daytona.io/
