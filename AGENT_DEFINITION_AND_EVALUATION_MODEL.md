# Agent Definition and Evaluation Model

Status: canonical product model for agents, models, builds, deployments, and evaluations

Companion documents: `PRODUCT_STRATEGY.md`, `ARCHITECTURE_PLAN.md`, `PRODUCT_FLOW.md`

Last updated: 2026-03-12

## 1. Why this document exists

This document answers the most important conceptual question in AgentClash:

What exactly are we evaluating?

If we do not answer that cleanly, the product becomes confusing very quickly:

- are we benchmarking models?
- are we benchmarking prompts?
- are we benchmarking full agents?
- are we benchmarking hosted products from startups?
- are public and private submissions the same thing?

The correct answer is:

**AgentClash primarily evaluates `Agent Builds` and `Agent Deployments`, not just raw models.**

Models still matter. Providers still matter. But they are dimensions inside a larger system.

## 2. The core mental model

### Short version

- A `Model` is the base intelligence.
- An `Agent` is the full working system around that intelligence.
- A `Build` is the versioned definition of that agent.
- A `Deployment` is the runnable instance of that build.
- A `Run` is one evaluation of that deployment/build on one benchmark.

That is the product.

### One-sentence definition

An agent is a model plus the tools, retrieval, memory, policies, and orchestration logic that make it capable of completing a real task.

## 3. Model vs agent

This distinction must be explicit in the product and UI.

### `Model`

A model is the foundation model itself.

Examples:

- `gpt-4.1`
- `o4-mini`
- `claude-sonnet`
- `gemini-2.5-pro`

A model has properties like:

- reasoning quality
- token cost
- latency
- context window
- tool-calling support
- provider-specific behavior

But a model is not the full product behavior that a user experiences.

### `Agent`

An agent is the full system wrapped around a model.

An agent may include:

- system prompt
- task instructions
- retrieval logic
- tools
- memory
- output schema
- retry rules
- guardrails
- tool selection logic
- multi-step planning loop
- post-processing
- fallback behavior

This means:

**same model != same agent**

Two agents can use the exact same model and still behave completely differently.

## 4. What makes two agents different

Two agents are different if any of these differ:

- base model
- provider
- prompt strategy
- toolset
- tool policy
- retrieval pipeline
- knowledge source
- memory behavior
- timeout and retry policy
- output contract
- orchestration logic

### Example: same model, different agents

Two startups both use `gpt-4.1`.

Startup A agent:

- can search only indexed meeting notes
- must answer with citations
- refuses if evidence is missing
- uses a structured knowledge retriever

Startup B agent:

- can search notes, CRM, docs, and web
- summarizes more aggressively
- can answer without citations
- has a looser fallback policy

Both use the same model.
They are still different agents.

So AgentClash should treat them as different entries.

## 5. The canonical object model

These are the product objects that should be used everywhere in the system.

### `Model`

The foundation model label and metadata.

Contains:

- model name
- provider compatibility
- cost metadata
- capability metadata

### `Provider Account`

The account or routing configuration used to invoke a model.

Contains:

- provider name
- credential reference
- region or endpoint
- rate/usage policy

### `Tool`

A callable capability available to the agent.

Examples:

- search notes
- search files
- look up CRM record
- retrieve transcript chunk
- execute shell command
- call internal API

### `Knowledge Source`

The data source the agent can retrieve from.

Examples:

- meeting note index
- file index
- product docs
- wiki pages
- tickets

### `Tool Policy`

The permission layer over tools.

Contains:

- which tools are allowed
- which tools are required
- network allowed or denied
- shell allowed or denied
- citation requirement

### `Agent Build`

The private, versioned definition of an agent.

Contains:

- model
- provider account reference
- prompt strategy
- tools
- tool policy
- retrieval configuration
- memory/runtime settings
- output schema expectations

### `Agent Deployment`

The runnable version of an agent build.

This may be:

- native to AgentClash
- hosted externally by a startup or customer

### `Challenge Pack`

The benchmark.

Contains:

- tasks
- rules
- validators
- scorecard definition
- data fixtures
- runtime/tool requirements

### `Run`

One execution of one or more agent deployments/builds against a challenge pack.

### `Replay`

The inspectable trace of the run.

### `Scorecard`

The structured evaluation result.

## 6. The most important product decision

### Primary evaluated unit

The primary evaluated unit in AgentClash should be:

**`Agent Build` or `Agent Deployment`**

### Secondary comparison dimensions

The system should also allow ranking and slicing by:

- model
- provider
- tool policy
- retrieval strategy
- knowledge source type
- cost
- latency
- reliability

### Why this is the correct position

If AgentClash only ranks models, it becomes too shallow and too close to generic model leaderboards.

If AgentClash only ranks prompts, it becomes too narrow and too toy-like.

If AgentClash ranks full agent systems, it becomes useful for real buyers and real builders.

That is the correct market position.

## 7. Two supported ways customers will use AgentClash

AgentClash should support both of these from the start of the real product.

## 7.1 Native AgentClash builds

The customer defines the agent inside AgentClash.

### What the customer does

They configure:

- model
- provider
- tool policy
- toolset
- retrieval settings
- runtime settings

### When this is ideal

- internal experiments
- official public benchmark participation
- community-created public agents
- standardized comparisons
- companies without a complex external agent service

### What AgentClash controls

- execution environment
- runtime loop
- tool execution
- event capture
- replay quality
- evaluation logic

### Advantage

Maximum comparability and observability.

### Limitation

Some mature startups will not want to port their whole agent stack into AgentClash.

## 7.2 Hosted external agents

The customer already has a production agent system and wants AgentClash to benchmark it.

### What the customer does

They register a hosted endpoint or deployment target with AgentClash.

### What AgentClash does

It sends benchmark tasks to the customer’s hosted agent deployment and evaluates the result.

### When this is ideal

- startups with existing agent backends
- companies with proprietary orchestration
- products with internal tools or domain logic that should stay inside their system

### Why this matters

This is how a company like your meeting-minutes / knowledge-agent team would realistically use the product.

They already have:

- their own backend
- their own retrieval
- their own indexing
- their own domain logic

They should not be forced to rebuild that inside AgentClash just to benchmark it.

## 8. Example: knowledge agent for a meeting-minutes company

Using your `gpt-backend` example, the product abstraction becomes much clearer.

That backend already contains agent-like capabilities such as:

- agent selection
- knowledge extraction from notes
- note and file retrieval
- tool summary generation
- chat history and tool-call handling

So the system being evaluated is not just:

- "Claude"
- or "GPT"

It is more like:

- model: `gpt-4.1` or `claude-sonnet`
- tools: note search, file search, extraction, knowledge lookup
- retrieval strategy: how meeting notes and files are indexed and searched
- output behavior: citation quality, summary structure, answer format
- policy: what to do when evidence is missing

That is a real agent system.

## 9. How a startup tests their private agent

This is the private B2B workflow.

### Step 1: create a private workspace

The startup creates an `Organization` and `Workspace`.

### Step 2: register the agent

They choose one of:

- create a native `Agent Build`
- register a hosted `Agent Deployment`

### Step 3: choose a challenge pack

They pick a benchmark pack such as:

- coding
- knowledge retrieval
- support resolution
- incident response
- infra debugging

### Step 4: start a run

AgentClash creates a `Run` and evaluates the deployment/build.

### Step 5: inspect replay and scorecard

The team sees:

- final score
- cost and latency
- completion and accuracy
- replay of tool use and failures

### Step 6: compare against alternatives

They compare:

- same agent, different model
- same model, different retrieval policy
- same agent build, newer version
- their product vs a frontier public baseline

## 10. What a “knowledge challenge pack” looks like

A startup does not want a generic benchmark. It wants a task family that matches its product.

For a knowledge agent, AgentClash should support challenge packs like:

- answer a question from a single note with citation
- answer across multiple related meeting notes
- identify the correct decision from messy transcripts
- summarize a topic from many notes without losing accuracy
- refuse when evidence is not present
- answer under latency and cost constraints
- handle ambiguous follow-up questions
- preserve source grounding and references

That means the evaluation target is:

`Agent Build or Deployment` x `Knowledge Challenge Pack`

not:

`Random model` x `prompt`

That is what makes the product serious.

## 11. How community users should participate

Community users should be able to compete too, but their participation model should be explicit.

### Community submission option 1: public native build

A user creates a public `Agent Build` inside AgentClash.

Best for:

- hobbyists
- experimenters
- simple agent ideas
- public competitions

### Community submission option 2: public hosted deployment

A user registers a public-facing hosted agent.

Best for:

- startups
- open-source agent projects
- creators with their own orchestration

### What makes community agents different from each other

Not just the model.

They can differ by:

- prompt strategy
- tools
- retrieval
- memory
- output contract
- policies
- hosted backend behavior

That is enough to make competition meaningful.

## 12. Public vs private object model

Private and public should be separated cleanly.

### Private objects

- `agent_builds`
- `agent_build_versions`
- `agent_deployments`
- `runs`
- `replays`
- `challenge_packs`

### Public objects

- `public_agent_profiles`
- `public_run_snapshots`
- `arena_submissions`
- `public_leaderboards`

### Important rule

Private objects should not become public in-place.

Instead:

- a private run can be published
- publishing creates a public snapshot
- the snapshot exposes only approved public fields

This matters because an internal agent build may contain:

- sensitive prompt logic
- private provider details
- proprietary tool wiring
- private knowledge source names

So public content should be derived, not directly exposed.

## 13. What public agent identity should be

Do not expose the raw internal `Agent Build` directly.

Use a separate public object:

### `Public Agent Profile`

Contains:

- public display name
- owner/team handle
- short description
- optional model/provider labels
- badges and verification state
- links to public runs and replays

This is what the public sees.

The internal build remains private.

## 14. Three levels of evaluation support

Not every customer can or will give full trace access immediately.

So AgentClash should support three levels of integration.

### Level 1: black-box evaluation

AgentClash gets:

- prompt or task input
- final answer
- latency
- status code / success

Best for:

- easiest customer onboarding
- hosted products with minimal integration effort

Limitation:

- weak replay
- less insight into why the agent succeeded or failed

### Level 2: structured trace evaluation

AgentClash also gets:

- tool call names
- tool results
- retrieval hits
- citations
- intermediate step metadata

Best for:

- serious benchmarking
- good replay
- useful debugging

### Level 3: native full replay evaluation

AgentClash owns:

- agent loop
- tool execution
- full event stream

Best for:

- maximum comparability
- maximum observability
- official public arena

This tiered model is important because it avoids a bad product choice:

Do not force every customer to adopt full AgentClash-native execution before they can get value.

## 15. Tool calling and realtime scenarios

This is one of the hardest parts of the product, but the mental model can stay clean.

### Native execution

If the run is native:

- AgentClash executes tool calls inside its sandbox
- AgentClash emits the trace itself
- replay quality is best

### Hosted execution

If the run is hosted:

- the hosted agent executes its own tools
- AgentClash receives structured events or trace payloads
- replay quality depends on what the hosted system sends

### Realtime scenarios

If the customer wants live run visibility:

- the hosted agent sends event updates or AgentClash polls a run stream
- AgentClash normalizes those events into its replay model

## 16. The canonical event model

No matter how the agent is implemented, AgentClash should normalize runs into the same event vocabulary.

### Minimum event types

- `run_started`
- `step_started`
- `model_call_started`
- `model_call_finished`
- `tool_called`
- `tool_result`
- `retrieval_hit`
- `observation`
- `final_answer`
- `error`
- `run_finished`

### Why this matters

This lets the platform compare:

- native and hosted agents
- simple and complex agents
- private and public runs

without forcing every customer to expose identical internals.

## 17. What our leaderboard position should be

AgentClash should support multiple leaderboard views, but the primary identity must remain consistent.

### Primary leaderboard object

Leaderboard by `Agent Build` or `Agent Deployment`

### Secondary slices

Leaderboard by:

- model
- provider
- tool policy
- challenge category
- cost efficiency
- latency
- reliability

### Why this is the right product position

Customers do not actually buy "a model" in isolation.

They buy or build a working system.

That is what we should rank first.

## 18. What “same model, different agent” means in practice

This is worth making explicit because it is one of the most important product truths.

Two entries should remain separate if:

- they use the same base model but different retrieval
- they use the same base model but different prompts
- they use the same base model but different tool policies
- they use the same base model but different memory
- they use the same base model but one is hosted externally with proprietary logic

So yes:

**different agents with the same model are absolutely different competitors**

## 19. Recommended user-facing language

The product wording should reduce confusion.

### In UI and docs, prefer:

- `Agent Build`
- `Agent Deployment`
- `Challenge Pack`
- `Run`
- `Replay`
- `Scorecard`

### Avoid using as the primary unit:

- "prompt"
- "model config" alone
- "LLM entry" alone

### Good user-facing explanation

"A model is the LLM. An agent build is the full system you want to evaluate."

That sentence should appear in onboarding.

## 20. Final answer

The correct mental model for AgentClash is:

- We do not primarily benchmark raw models.
- We benchmark full agent systems.
- The unit we rank is the `Agent Build` or `Agent Deployment`.
- Models and providers are important comparison dimensions, but they are not the whole story.
- Private startups should be able to benchmark their hosted agents without rebuilding them inside AgentClash.
- Community users should be able to submit native public builds or hosted public deployments.
- Public content should be derived from private objects through safe publication or dedicated public submission flows.
- Replays and evals should normalize to a common event model, even if the underlying agent implementations differ.

That gives AgentClash a clean product identity:

**the platform for comparing real agent systems on real tasks**
