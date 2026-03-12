# AgentClash V1 Requirements and Tech Spec

Status: canonical v1 product and implementation spec

Companion documents: `product-strategy.md`, `architecture-plan.md`, `product-flows.md`, `agent-evaluation-model.md`

Last updated: 2026-03-12

## 1. Purpose

This document defines exactly what AgentClash v1 is.

It is the bridge between strategy and implementation. It should be detailed enough that a product team and engineering team can build v1 without re-deciding the fundamentals.

This document answers:

- what v1 must do
- who v1 is for
- what is in scope and out of scope
- what the product objects are
- what the frontend and backend must support
- what the API and data model must look like
- what the minimum technical architecture is
- what counts as “done” for v1

## 2. V1 definition

### V1 one-line definition

AgentClash v1 is a cloud product where teams can benchmark private agent systems on official engineering challenge packs, watch live runs, inspect replays, compare scorecards, and optionally publish selected results into a curated public arena.

### V1 product shape

V1 has two connected surfaces:

- `Private Workspace Beta`
- `Public Arena Alpha`

The private workspace is the primary value and revenue surface.
The public arena is a smaller but deliberate distribution and credibility surface.

### V1 business goal

Prove that teams will pay for:

- replayable agent evaluation
- challenge-pack-based benchmarking
- side-by-side comparison across agent builds and providers

### V1 product goal

Make one workflow excellent:

`Create or register an agent -> run it on an official benchmark -> inspect replay -> compare results -> make a decision`

## 3. Who v1 is for

### Primary user

AI product teams and applied AI teams building engineering-related agents.

Examples:

- coding agents
- debugging agents
- incident-response agents
- knowledge agents over internal docs or meeting notes
- infra/configuration agents

### Primary v1 customer profile

Teams that already have either:

- an internal agent they want to compare
- multiple model/provider choices they want to evaluate
- a need for proof before they ship or buy an agent system

### Secondary v1 user

Public viewers who want to browse official challenge leaderboards and inspect selected public runs.

### Explicit non-v1 customer

- broad consumer audiences looking for a fun toy
- general no-code automation teams
- non-engineering agent builders as the main audience

## 4. V1 goals and non-goals

### V1 goals

1. Let a team create a workspace and register one or more agent systems.
2. Let that team run official challenge packs against those agents.
3. Produce live run telemetry, final scorecards, and replay pages.
4. Support both native AgentClash builds and hosted external agents.
5. Allow selective publication of completed runs into a public arena.
6. Enforce basic billing, quota, and workspace permissions.

### Non-goals

1. Full self-serve user-authored challenge packs.
2. Open community submissions with unlimited untrusted agents.
3. Self-hosted deployment.
4. Arbitrary workflow-builder features.
5. Highly customizable tournament/league mechanics.
6. A separate provider marketplace or model proxy business.

## 5. V1 scope boundaries

### In scope

- official challenge packs for engineering tasks
- workspace/org model
- native agent builds
- hosted agent deployment benchmarking
- run lifecycle and live telemetry
- replay and scorecards
- curated public arena
- publish-from-private workflow
- billing and usage limits

### Out of scope

- self-serve private challenge pack authoring
- open public user-generated challenge ecosystem
- bring-your-own sandbox runtime
- enterprise compliance pack beyond baseline controls
- full CI/CD benchmark automation for every customer
- cross-workspace data warehouse analytics

## 6. V1 product objects

These objects are required in v1 and must remain consistent across backend, frontend, and docs.

### `Organization`

Billing and admin boundary.

### `Workspace`

Operational boundary for runs, builds, and challenge access.

### `Challenge Pack`

Official benchmark family, such as:

- coding fixes
- infra config debugging
- incident investigation
- knowledge retrieval and grounded QA

### `Challenge Pack Version`

Immutable version of the benchmark used for real runs and leaderboard comparability.

### `Agent Build`

Private, versioned agent definition managed inside AgentClash.

### `Agent Deployment`

Runnable agent endpoint.

Two supported deployment types in v1:

- `native`
- `hosted_external`

### `Run`

One benchmark execution against one challenge pack version.

### `Run Agent`

One participating agent entry inside a run.

### `Replay`

The structured timeline of run events.

### `Scorecard`

Structured performance summary for a run agent and for the whole run.

### `Public Agent Profile`

Public-facing profile used in the curated arena.

### `Public Run Snapshot`

Published, sanitized derivative of a private run.

## 7. V1 user stories

### Workspace setup

- As a team admin, I can create an organization and workspace.
- As a team admin, I can invite teammates into the workspace.
- As a workspace user, I can see benchmark packs available to my workspace.

### Native build evaluation

- As a user, I can define a native agent build with a model, provider, tool policy, and runtime settings.
- As a user, I can run that build against an official challenge pack.
- As a user, I can watch live progress and inspect the replay afterward.

### Hosted agent evaluation

- As a user, I can register an externally hosted agent deployment.
- As a user, I can benchmark that external deployment against an official challenge pack without rebuilding it in AgentClash.
- As a user, I can compare hosted external runs against native builds or other hosted agents.

### Analysis

- As a user, I can compare two or more runs and see score differences.
- As a user, I can inspect tool/retrieval behavior where trace support exists.
- As a user, I can distinguish between model effects and full agent-system effects.

### Public arena

- As a viewer, I can browse official public leaderboards.
- As a workspace user, I can selectively publish a completed run to the public arena.
- As a viewer, I can inspect a published public replay snapshot.

## 8. V1 product requirements

## 8.1 Authentication, organizations, and workspaces

### Requirements

- Users must authenticate through WorkOS.
- A user must belong to at least one organization to use the private product.
- Every private run must belong to exactly one workspace.
- Workspace membership and role must gate:
  - creating agent builds
  - starting runs
  - viewing replays
  - publishing results

### Roles for v1

- `org_admin`
- `workspace_admin`
- `workspace_member`
- `workspace_viewer`

### Acceptance criteria

- Unauthorized users cannot start or inspect private runs.
- A user in one workspace cannot access another workspace’s runs.

## 8.2 Challenge packs

### V1 challenge-pack policy

Only official challenge packs are self-serve in v1.

V1 supports these families:

- coding
- infra/config
- incident response
- knowledge retrieval and grounded QA

### Challenge-pack requirements

Each challenge pack version must define:

- benchmark metadata
- task list
- scoring rules
- runtime/tool policy
- visibility policy
- leaderboard eligibility rules

### Knowledge-agent requirement

V1 must include at least one official knowledge-agent pack for grounded QA or retrieval-oriented systems.

This is required because startups with knowledge agents are part of the target customer set.

### Out of scope

- end-user self-serve private challenge authoring

### Allowed compromise for v1

AgentClash team can manually onboard partner/private benchmark variants behind internal tooling, but that is not a general user-facing product surface in v1.

## 8.3 Agent builds

### Native agent builds

V1 native agent builds must support:

- model selection
- provider account selection
- tool policy selection
- runtime/timeout profile
- versioning

### Required fields

- display name
- description
- deployment type
- model
- provider account reference
- tool policy
- runtime profile

### V1 simplification

Do not expose arbitrary free-form agent graph builders in v1.

Instead, native builds are configuration-driven:

- model
- tools
- policy
- runtime settings

### Acceptance criteria

- A user can create a native build and use it in a run without touching raw config files.

## 8.4 Hosted external agent deployments

This is a critical v1 requirement.

### Why it is required

Real startups already have agent systems. They will not all rebuild those systems inside AgentClash.

### Hosted deployment requirements

Users must be able to register:

- deployment name
- base URL
- auth secret reference
- supported trace level
- timeout profile

### Supported hosted execution modes in v1

- `black_box`
- `structured_trace`

### `black_box` mode

AgentClash sends a benchmark task and receives:

- final output
- success or failure
- latency
- optional metadata

### `structured_trace` mode

AgentClash also receives structured events such as:

- tool calls
- retrieval hits
- intermediate status
- citations

### V1 hosted integration contract

#### Start request from AgentClash to customer deployment

`POST /agentclash/runs`

Payload must include:

- `run_id`
- `run_agent_id`
- `challenge_pack_version_id`
- `task_payload`
- `trace_level`
- `callback_url`
- `deadline_at`

#### Minimum synchronous response

- `accepted`
- `external_run_id`

#### Callback contract from hosted agent back to AgentClash

`POST /v1/integrations/hosted-runs/:run_id/events`

Allowed event types:

- `run_started`
- `model_call_started`
- `model_call_finished`
- `tool_called`
- `tool_result`
- `retrieval_hit`
- `final_answer`
- `error`
- `run_finished`

### Important v1 decision

Do not support arbitrary customer-specific protocols.
There must be one AgentClash hosted-agent contract.

## 8.5 Run execution

### Run initiation requirements

When a user clicks `Start Run`, the system must:

1. validate access and billing
2. create run records
3. start a durable workflow
4. route each participating agent into execution

### Required run states

- `queued`
- `starting`
- `running`
- `scoring`
- `completed`
- `failed`
- `cancelled`

### Agent run states

- `queued`
- `provisioning`
- `running`
- `completed`
- `failed`
- `timed_out`

### Supported v1 run types

- single-agent benchmark
- multi-agent comparison run

### V1 concurrency rule

Each workspace has plan-based concurrency limits.

## 8.6 Native execution and sandboxing

### Native execution requirements

Native builds must execute in isolated sandboxes, not on the API or worker host filesystem directly.

### V1 sandbox requirements

- isolated workspace per agent run
- challenge pack assets mounted or uploaded into sandbox
- tool execution restricted by policy
- outbound network disabled by default

### Tool policy requirements

Each challenge pack version must define allowed tool classes.

V1 supported tool classes:

- file read/write/list
- search
- structured build/test or task-specific tools
- optional shell only where explicitly allowed

### Important decision

V1 native sandbox provider is `E2B`.

## 8.7 Replay

Replay is a required product surface, not a debug bonus.

### Replay requirements

Each run agent must have a replay page containing:

- step timeline
- state transitions
- model call summaries
- tool call summaries
- errors
- final output
- score summary

### Replay data levels

V1 must support:

- full replay for native runs
- partial replay for hosted black-box runs
- structured replay for hosted structured-trace runs

### Acceptance criteria

- After run completion, the replay is available without requiring logs from the worker host.

## 8.8 Scorecards

### Required score dimensions in v1

- completion
- correctness
- speed
- cost
- reliability

### Challenge-specific metrics

Challenge packs may define additional metrics such as:

- citation fidelity
- refusal correctness
- recovery quality
- benchmark-specific pass/fail rules

### Scorecard requirements

The user must be able to see:

- total score
- component breakdown
- run duration
- token/cost data where available
- status of scoring confidence

## 8.9 Compare

### Compare requirements

Users must be able to compare at least two runs and see:

- score differences
- completion differences
- cost differences
- latency differences
- replay summary differences

### V1 simplification

Do not implement arbitrary large-scale analytics dashboards yet.
Keep compare focused on selected runs.

## 8.10 Public arena alpha

### Public arena requirements

The public arena in v1 is curated and official-first.

It must support:

- public leaderboard pages
- public challenge pages
- public run pages
- public replay snapshots
- public agent profiles

### V1 public participation rule

V1 public arena does **not** support unrestricted open community submissions by default.

Allowed public content in v1:

- official benchmark runs
- workspace runs explicitly published by authorized users
- approved/curated partner submissions

### Why

This protects public leaderboard credibility and reduces moderation complexity.

## 8.11 Publication flow

### V1 publication rule

Private by default.

### Publish action

A workspace user with permission can publish a completed run.

Publishing creates:

- `public_agent_profile` if needed
- `public_run_snapshot`
- public replay snapshot
- leaderboard eligibility record if applicable

### Important decision

Private objects never become public in-place.
Public objects are derived snapshots.

### User-controlled publish options

- public display name
- show team name or anonymous
- show model/provider or hide details
- show replay summary only or full public replay

## 8.12 Billing and quotas

### V1 billing requirements

The system must enforce:

- workspace plan
- run quota
- concurrency limit
- retention policy

### Required plan checks

- before run creation
- before export or premium actions later

### Billing system

- Stripe as billing source
- PostgreSQL as usage and entitlement materialization

## 9. V1 non-functional requirements

### Reliability

- no run should disappear once accepted
- replay and scorecard state must survive worker restarts

### Security

- provider credentials encrypted
- sandboxed native execution
- private-by-default data visibility

### Performance

- live run page should update within low seconds, not minutes
- public leaderboard pages should load fast enough for normal web use

### Observability

- every run must have traceable workflow state
- worker failures must be visible in ops tooling

### Scalability

V1 should scale to:

- hundreds of workspaces
- thousands of runs
- modest concurrent execution

without redesigning the core architecture

## 10. V1 technical architecture

## 10.1 Frontend

### Stack

- Next.js 15
- React 19
- TypeScript 5
- Tailwind CSS 4
- TanStack Query

### Responsibilities

- public arena pages
- workspace app UI
- auth session handling
- live run screen
- replay UI
- compare UI
- billing UI

### Required pages

- landing page
- public challenge pack page
- public leaderboard page
- public run page
- workspace dashboard
- agent builds page
- agent deployments page
- challenge pack browser
- start-run page
- live run page
- replay page
- compare page
- settings/billing page

## 10.2 Backend services

### Required runtime services

- `web app`
- `api-server`
- `worker`

### Required managed services

- PostgreSQL
- Redis
- Temporal
- S3
- WorkOS
- Stripe
- E2B

### Service responsibilities

#### `api-server`

Owns:

- authz
- CRUD
- run creation
- publication flow
- replay retrieval
- billing enforcement
- public arena data serving

#### `worker`

Owns:

- native sandbox execution
- hosted deployment orchestration
- provider calls
- event capture
- artifact storage
- scorecard finalization

## 10.3 Deployment topology

### Production

- `Vercel` for web app
- `AWS ECS/Fargate` for api-server and worker
- `RDS Postgres`
- `ElastiCache Redis`
- `S3`
- `Temporal Cloud`
- `WorkOS`
- `Stripe`
- `E2B`

### Why this is locked

V1 should use managed-heavy infrastructure to optimize for product speed, not infra invention.

## 10.4 Core database schema

### Identity

- `users`
- `organizations`
- `organization_memberships`
- `workspaces`
- `workspace_memberships`

### Challenge system

- `challenge_packs`
- `challenge_pack_versions`
- `challenge_tasks`
- `scorecard_definitions`

### Agent system

- `provider_accounts`
- `agent_builds`
- `agent_build_versions`
- `agent_deployments`
- `tool_policies`

### Runs

- `runs`
- `run_agents`
- `run_steps`
- `run_events_index`
- `run_artifacts`
- `run_scorecards`

### Public objects

- `public_agent_profiles`
- `public_run_snapshots`
- `leaderboards`
- `leaderboard_entries`

### Billing

- `billing_accounts`
- `billing_subscriptions`
- `usage_counters`

## 10.5 Data storage rules

### PostgreSQL stores

- canonical product records
- run metadata
- scorecards
- replay indexes
- public arena metadata

### S3 stores

- large trace payloads
- large tool outputs
- exports
- replay artifacts

### Redis stores

- live event fanout
- short-lived state
- cache

## 10.6 API surface

### Required REST groups

- `/v1/me`
- `/v1/organizations`
- `/v1/workspaces`
- `/v1/challenge-packs`
- `/v1/agent-builds`
- `/v1/agent-deployments`
- `/v1/runs`
- `/v1/replays`
- `/v1/compare`
- `/v1/public`
- `/v1/billing`
- `/v1/integrations/hosted-runs`

### Required realtime endpoint

- `/v1/runs/:id/live`

Transport:

- WebSocket

### Required write endpoints

- create workspace
- create agent build
- create hosted deployment
- start run
- publish run

## 10.7 Realtime event model

### Minimum normalized event types

- `run_started`
- `step_started`
- `model_call_started`
- `model_call_finished`
- `tool_called`
- `tool_result`
- `retrieval_hit`
- `final_answer`
- `error`
- `run_finished`

### Event flow

- worker emits event
- Redis fans out live updates
- API server pushes to connected WebSocket clients
- important event summaries are persisted to PostgreSQL

## 10.8 Workflow orchestration

### Orchestration engine

- Temporal

### Parent workflow

- `RunWorkflow`

Responsibilities:

- initialize run
- create child workflows for each run agent
- handle deadline and cancellation
- finalize scoring
- finalize public materialization if already queued

### Child workflow

- `RunAgentWorkflow`

Responsibilities:

- prepare execution
- execute native sandboxed run or hosted run orchestration
- collect artifacts
- persist events

## 10.9 Native vs hosted execution rules

### Native

- full replay
- full sandbox control
- full tool-policy enforcement

### Hosted

- AgentClash does not run the agent internals
- AgentClash evaluates outputs and accepted traces
- replay quality depends on integration level

### Important v1 rule

Hosted agents must still normalize into AgentClash’s event model.

## 10.10 Score computation

### v1 scorecard formula model

Each challenge pack version defines:

- weighted metrics
- pass/fail rules
- leaderboard eligibility

### Required standard metrics

- completion
- correctness
- speed
- cost
- reliability

### Materialization rule

Scorecards are materialized and stored after run completion.
Leaderboards read materialized entries, not ad hoc recomputation.

## 11. UX requirements by major screen

## 11.1 Workspace dashboard

Must show:

- recent runs
- agent builds
- available challenge packs
- quota or plan status
- empty state guidance

## 11.2 Start run screen

Must let user:

- choose challenge pack version
- choose one or more builds/deployments
- see estimated cost/concurrency impact later if available
- start the run

## 11.3 Live run screen

Must show:

- run status
- participating agents
- live event timeline
- per-agent state
- partial counters

## 11.4 Replay page

Must show:

- score summary
- step timeline
- tool/retrieval events where available
- final output
- errors

## 11.5 Compare page

Must show:

- score breakdown side by side
- latency/cost/completion comparison
- replay summary diff

## 11.6 Public run page

Must show:

- public snapshot metadata
- public scorecard
- allowed replay view
- challenge context

## 12. Acceptance criteria for v1 launch

V1 is launch-ready when all of the following are true.

### Product acceptance

1. A new workspace can be created end to end.
2. A user can create a native agent build.
3. A user can register a hosted external deployment.
4. A user can start a run against an official challenge pack.
5. The user can watch live progress on the run page.
6. After completion, the replay is accessible.
7. After completion, a scorecard is visible.
8. The user can compare two runs.
9. The user can publish a selected run into the public arena.
10. A public visitor can view official/public leaderboard and replay pages.

### Technical acceptance

1. Run state survives worker restart.
2. Replay artifacts remain available after run completion.
3. Native runs execute inside isolated sandboxes.
4. Hosted runs can complete through the standardized integration contract.
5. Billing and quota checks prevent unauthorized overuse.
6. Workspace authorization prevents data leakage across tenants.

## 13. V1 implementation phases

## Phase 1: core private workspace

- auth/org/workspace
- official challenge packs
- native agent builds
- run creation and execution
- replay and scorecards

## Phase 2: hosted deployment support

- hosted deployment registration
- hosted run contract
- black-box mode
- structured trace mode

## Phase 3: public arena alpha

- public pages
- publish flow
- curated leaderboards
- public run snapshots

## Phase 4: billing and hardening

- Stripe entitlements
- quotas
- role polish
- observability hardening

## 14. Explicit v1 decisions

These are locked for v1.

- Private by default.
- Official challenge packs first.
- Hosted external agents are required.
- Open community submissions are deferred.
- Public arena is curated alpha, not a free-for-all.
- Agent Build / Deployment is the primary unit of evaluation.
- Public content is always a derived snapshot, not direct exposure of private objects.
- Native execution uses E2B-backed sandboxing.
- Temporal is the workflow engine.
- PostgreSQL is the source of truth.
- REST + WebSocket is the external application interface.

## 15. Final v1 statement

AgentClash v1 is not a toy race demo and not a generic model leaderboard.

It is a private benchmarking product for real agent systems, with enough public arena functionality to build trust and distribution. It supports both native builds and hosted external agents, official challenge packs, live runs, replay, scorecards, compare, and selective publication.

If those pieces work together cleanly, v1 succeeds.
