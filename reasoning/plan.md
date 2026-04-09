# agentclash-reasoning: v0 Implementation Plan

> Rewrite of the earlier exploratory draft for [agentclash/agentclash#118](https://github.com/agentclash/agentclash/issues/118).
>
> This document is the implementation plan to build the first `reasoning_v1` lane in AgentClash. It intentionally narrows scope to the smallest end-to-end slice that fits the current Go codebase and the decisions made during review.

---

## Table of Contents

1. [Goal](#1-goal)
2. [Codebase Constraints](#2-codebase-constraints)
3. [What Changed and Why](#3-what-changed-and-why)
4. [v0 Scope and Non-Goals](#4-v0-scope-and-non-goals)
5. [Architecture and Ownership](#5-architecture-and-ownership)
6. [Routing and Frozen Execution Context](#6-routing-and-frozen-execution-context)
7. [Bridge Contract](#7-bridge-contract)
8. [Canonical Event Contract](#8-canonical-event-contract)
9. [Go Workflow and Control Plane](#9-go-workflow-and-control-plane)
10. [Python Runtime Plan](#10-python-runtime-plan)
11. [Guardrails and Tool Execution](#11-guardrails-and-tool-execution)
12. [Persistence, Replay, and Scoring](#12-persistence-replay-and-scoring)
13. [Failure Handling and Recovery](#13-failure-handling-and-recovery)
14. [Delivery Plan](#14-delivery-plan)
15. [Success Criteria](#15-success-criteria)

---

## 1. Goal

Ship one new execution lane, `reasoning_v1`, that:

- runs a Python-managed ReAct loop,
- uses Go as the only tool-execution and sandbox authority,
- emits canonical AgentClash `run_events`,
- works with existing replay and scoring with only a small replay-builder addition,
- is safe to ship behind a single global enable flag.

This is not a research document anymore. It is a delivery plan for a constrained v0.

---

## 2. Codebase Constraints

The plan is shaped by what already exists in the repo today.

### Frozen execution context already exists

`RunAgentExecutionContext` in `backend/internal/repository/run_agent_execution_context.go` already gives us:

- frozen deployment snapshot data,
- provider wiring,
- runtime profile limits,
- challenge manifest and inputs,
- output schema,
- run and run-agent identifiers.

Reason this matters: v0 should consume that frozen execution snapshot directly instead of inventing a second prompt/tool/model expansion layer in Go.

### Hosted execution is a black-box callback lane

The current hosted path is:

- one start activity,
- one callback route,
- one execution record,
- one workflow signal type,
- terminal-event driven wakeups.

Reason this matters: reasoning needs nonterminal actionable events, live proposal state, sandbox ownership, and stricter protocol validation. It should be a separate lane, not an overload of hosted runs.

### Canonical events already drive replay and scoring

`backend/internal/runevents/envelope.go`, `backend/internal/repository/run_agent_replay_builder.go`, and `backend/internal/scoring/engine.go` already assume canonical `run_events`.

Reason this matters: v0 should keep the existing canonical event taxonomy and payload shapes where possible, instead of introducing a second reasoning-specific event vocabulary.

### Event storage is append-only by DB-assigned sequence

`run_events` persists one event at a time and assigns `sequence_number` inside Go using append-only semantics.

Reason this matters: v0 should keep a single event producer for the reasoning lane and avoid Go-side callback dedupe / out-of-order protocols that the schema does not support yet.

### Native execution already owns the real security boundary

Go native execution already owns:

- tool registry,
- tool allowlist enforcement,
- sandbox policy,
- shell/network gating,
- argument validation at execution time.

Reason this matters: Python can own reasoning strategy, but not tool enforcement.

---

## 3. What Changed and Why

This section captures the major changes from the earlier draft and the reason for each one.

| Change | New Plan | Reason |
|---|---|---|
| Runtime boundary | Go owns tool execution and run durability; Python owns reasoning strategy and model calls only | Avoid duplicating sandbox policy, guardrails, and trace semantics across two runtimes |
| Execution lane | Add a separate `reasoning_v1` lane | Hosted runs are a black-box adapter and do not fit actionable nonterminal events |
| Canonical event taxonomy | Keep existing `system.*`, `model.*`, `tool.*`, `sandbox.*` event types | Replay and validation already key directly off existing canonical types |
| Callback dedupe | No Go-side callback dedupe or gap detection in v0 | Current schema does not support the proposed idempotency model cleanly |
| Stored deployment type | Persist reasoning builds as `deployment_type="native"` and `execution_target="native"` | Reasoning is an internal alternate executor, not a hosted customer endpoint |
| Event source | Add `source="reasoning_engine"` | We need to distinguish Python-lane events from Go-native events without inventing new event types |
| Routing | Route only when `REASONING_SERVICE_ENABLED=true` and frozen `agent_kind=="reasoning_v1"` | v0 should not add workspace allowlists, rollout percentages, or persisted `execution_lane` |
| Sandbox lifecycle | One sandbox per run, created lazily on first tool batch, then reattached by `sandbox_id` | Preserves cross-tool filesystem state without paying eager setup cost for tool-free runs |
| Workflow wakeups | Persist every callback, but signal Temporal only for tool proposals and terminal events | Temporal should remain a coordinator, not a progress bus |
| Tool handoff | Proposal/result exchange is batched per model turn | Matches native semantics and preserves ordered same-turn tool dependencies |
| Start payload | `POST /reasoning/runs` receives a typed `execution_context` blob | Avoids duplicating prompt/tool/model derivation logic in Go and Python |
| Summary metadata expansion | Keep reasoning-specific metadata in `payload`, not new persisted summary fields | `run_events` persistence does not store arbitrary extra summary fields today |
| Success terminal events | Successful runs emit both `system.output.finalized` and `system.run.completed` | Keeps answer-finalization timing distinct from run completion timing |
| Execution state row | Add `reasoning_run_executions` keyed by `run_agent_id` | `run_events` is not a mutable control-plane state table |
| Guardrail split | Python handles pre-run and final-output validation; Go enforces pre-tool and post-tool execution boundaries | Security-critical enforcement must happen at the executor boundary |
| Temporal model | Use short activities plus workflow signal-wait loop only | A heartbeat long-running activity adds complexity without fitting current Temporal patterns |
| Credentials | Go resolves provider secrets at start and passes Python usable credentials in the trusted payload | No scoped-token broker or Go model-proxy layer exists yet |
| Streaming | No live token streaming in v0 | There is no Redis/WebSocket transport in the repo today |
| Eval harness | Reuse existing Go scoring and add reasoning-specific trace tests only | A second evaluation stack is too large for the first runtime slice |
| Strategy scope | `react` only in v0 | `plan_execute` and `reflect_once` materially expand control-flow complexity |
| Service endpoint config | Reasoning base URL is worker/global config, not per-deployment DB state | Reasoning remains internally routed native execution |
| Callback payloads | Python sends canonical `runevents.Envelope` events directly | We control both sides of the bridge, so a second normalization layer is unnecessary |
| Python runtime substrate | Build a first-party loop and model client; do not depend on `pydantic-ai` runtime orchestration | `Agent.run()` does not fit an external Go-owned tool loop |
| Provider scope | OpenAI-compatible `/chat/completions` only in v0 | This matches current native provider support and keeps parity tight |
| Payload parity | Keep native payload shapes for `model.call.*` and `tool.call.*` wherever possible | Existing replay/scoring already understand them |
| Event identity | Treat `event_id` as opaque; Go assigns persisted `sequence_number` | Removes a second ordering channel that Go would not actually use |
| Callback route | Add a dedicated reasoning callback route and ingestion manager | Avoid polluting hosted ingestion abstractions |
| Tool policy source | Challenge manifest plus `RuntimeProfile` remain the source of truth for tool policy | Avoid duplicated allowlists in `guardrail_spec` |
| Output schema source | Final-output validation uses the frozen `output_schema` only | Avoid two separate schema pointers for the same answer boundary |
| Tool registry | Reuse Go native sandbox tools minus `submit` | Keep reasoning capabilities aligned with what Go can actually execute |
| Trace detail | Emit terminal tool events only, no `tool.call.started` or granular `sandbox.*` trace expansion in v0 | Keeps instrumentation work aligned with native parity |
| Repair loop | Final-output repair retries are real additional no-tool steps | They consume real model budget and should appear in the trace |
| Proposal state | Persist one current pending proposal on `reasoning_run_executions` | Lets the workflow load actionable state after a minimal signal |
| Terminal stream rules | Freeze the stream after terminal; only exact terminal retries are accepted | Prevents replay/scoring corruption from post-terminal appends |
| Protocol violations | Fail the reasoning execution immediately in control-plane state | Deterministic protocol failures should not degrade into timeouts |

---

## 4. v0 Scope and Non-Goals

### In Scope

- One new `agent_kind`: `reasoning_v1`
- One strategy: `react`
- One Python runtime loop
- One Go callback route and ingestion manager
- One new execution-state table: `reasoning_run_executions`
- OpenAI-compatible non-streaming model calls only
- Canonical event emission for replay/scoring
- Go-owned sandbox tool execution
- Deterministic guardrails

### Explicitly Out of Scope

- `plan_execute`
- `reflect_once`
- `pydantic-ai` runtime orchestration
- live token streaming
- Redis / WebSocket fanout
- separate Python eval harness
- per-workspace rollout config
- percentage rollout
- persisted `execution_lane`
- resumable runs
- MCP, A2A, memory, multi-agent work
- rich `sandbox.*` tracing
- provider support beyond OpenAI-compatible chat completions

Reason: v0 should prove one clean bridge and one clean control loop before adding strategy or platform breadth.

---

## 5. Architecture and Ownership

### Go owns

- run/workflow orchestration,
- routing,
- callback auth,
- callback validation,
- event persistence,
- mutable execution state,
- tool execution,
- sandbox lifecycle,
- tool policy enforcement,
- post-tool sanitization,
- final cleanup.

Reason: these are the durability and trust-boundary responsibilities already concentrated in Go.

### Python owns

- the ReAct loop,
- model message assembly from frozen execution context,
- provider request construction,
- final-answer candidate detection,
- pre-run input checks,
- final-output validation and retry,
- callback emission in canonical order.

Reason: these are reasoning-strategy responsibilities and do not require ownership of the sandbox or database.

### Ownership boundary

The Python service is not another executor. It is a reasoning coordinator that pauses whenever tool execution is needed and resumes only after Go returns the authoritative sanitized tool results.

---

## 6. Routing and Frozen Execution Context

### Routing rule

Route to the reasoning lane only when both are true:

1. global worker config `REASONING_SERVICE_ENABLED=true`
2. frozen `executionContext.Deployment.AgentBuildVersion.AgentKind == "reasoning_v1"`

Reason: v0 should route by explicit build intent plus one global kill switch only.

### Stored model

`reasoning_v1` builds remain stored as:

- `deployment_type = "native"`
- `runtime_profiles.execution_target = "native"`

Reason: the DB model already understands native vs hosted; reasoning is an internal alternate native executor.

### Service endpoint configuration

The reasoning service base URL is worker/global configuration, not per-deployment database state.

Reason: the reasoning lane is internally routed native execution, not a deployment-level customer endpoint.

### Frozen source of truth

Every routing and runtime decision must come from the frozen `RunAgentExecutionContext`, not mutable current build rows.

Reason: retries must stay deterministic even if build definitions change later.

### New DB/API work

- add `"reasoning_v1"` to the `agent_kind` DB constraint
- add `"reasoning_v1"` to API validation

Reason: non-empty `reasoning_spec` is too ambiguous to use as a routing signal.

### Trace mode rule

`reasoning_v1` should reject `runtime_profile.trace_mode = "disabled"`.

`"preferred"` and `"required"` are treated the same in v0: the lane always emits the mandatory structured event set.

Reason: reasoning control flow depends on structured callbacks, so a true trace-disabled mode is not compatible with the lane.

---

## 7. Bridge Contract

The bridge stays HTTP/JSON and callback-based.

### Start endpoint

`POST /reasoning/runs`

Request shape:

```json
{
  "run_id": "uuid",
  "run_agent_id": "uuid",
  "idempotency_key": "string",
  "execution_context": {},
  "callback_url": "string",
  "callback_token": "string",
  "deadline_at": "timestamp"
}
```

`execution_context` includes the frozen deployment/runtime snapshot, output schema, challenge inputs, and Go-resolved provider credentials needed for Python model calls.

Response shape:

```json
{
  "accepted": true,
  "reasoning_run_id": "string"
}
```

Rules:

- idempotent by `(run_agent_id, idempotency_key)`
- duplicate start returns the original accepted response and `reasoning_run_id`
- `accepted: true` means Python has durably recorded the start state
- synchronous start rejection produces no canonical `run_events`

Reason: Go needs a durable external acceptance point, but a blocked or malformed start should not fabricate a runtime trace.

### Tool-results endpoint

`POST /reasoning/runs/{id}/tool-results`

Request shape:

```json
{
  "idempotency_key": "proposal_event_id",
  "tool_results": [
    {
      "tool_call_id": "string",
      "status": "completed | blocked | skipped | failed",
      "content": "string",
      "error_message": "string?"
    }
  ]
}
```

Rules:

- idempotent by `(reasoning_run_id, idempotency_key)`
- the idempotency key is the triggering proposal `event_id`
- `accepted: true` means Python durably recorded the batch
- Go retries submit only; it never reruns an already executed tool batch

Reason: tool execution has real side effects, so submit retries must be independent from tool retries.

### Cancel endpoint

`POST /reasoning/runs/{id}/cancel`

Request shape:

```json
{
  "idempotency_key": "string",
  "reason": "string"
}
```

Response shape:

```json
{
  "acknowledged": true
}
```

Rules:

- idempotent by `(reasoning_run_id, idempotency_key)`
- `acknowledged: true` means Python durably recorded the cancel request
- cancellation is control-plane state, not a required canonical `system.run.failed`

Reason: user cancellation should not be modeled as an ordinary runtime failure.

### Callback auth

Reuse the existing signed bearer-token pattern used by hosted callbacks.

Reason: the repo already has a working callback-auth primitive; v0 does not need a second auth subsystem.

### Callback route

Add a dedicated route for reasoning events, separate from hosted runs.

The handler accepts exactly one canonical event per HTTP request.

Reason: hosted ingestion is typed around hosted normalization and `hosted_run_executions`, which do not fit the reasoning lane.

---

## 8. Canonical Event Contract

### Event envelope

The callback payload is a canonical `runevents.Envelope` event.

Rules:

- `source` is required and must equal `"reasoning_engine"`
- `evidence_level` is `"native_structured"`
- `event_id` is opaque and unique
- Go ignores any Python-side `sequence_number`; DB persistence assigns sequence order
- no top-level callback `idempotency_key`
- no `reasoning_run_id` inside canonical events

Reason: the reasoning lane should emit the same canonical event shape the rest of AgentClash already understands.

### Canonical event set in v0

The required v0 event vocabulary is:

- `system.run.started`
- `system.step.started`
- `model.call.started`
- `model.call.completed`
- `model.tool_calls.proposed`
- `tool.call.completed`
- `tool.call.failed`
- `system.step.completed`
- `system.output.finalized`
- `system.run.completed`
- `system.run.failed`

No new canonical `reasoning.*` event types.
No `model.output.delta` events in v0.

Reason: replay and validation already understand this vocabulary.

### Required payload parity

#### `system.run.started`

Must keep native-compatible payload values:

- `deployment_type = "native"`
- `execution_target = "native"`

Reason: route identity belongs in `source`, not in a second payload-level execution taxonomy.

#### `model.call.completed`

Must mirror native payload shape as closely as possible:

- `provider_key`
- `provider_model_id`
- `finish_reason`
- `output_text`
- `tool_calls`
- `usage.{input_tokens,output_tokens,total_tokens}`
- `raw_response`

No v0-only additions such as:

- `usage.details`
- `streaming`
- `latency_ms`
- `estimated_cost_usd`

Reason: existing replay/scoring consumers already understand the native payload shape.

#### `tool.call.*`

Must mirror native parity for real executions, with one explicit extension for blocked-before-execution failures:

- top-level `tool_call_id`
- top-level `tool_name`
- top-level `arguments`
- nested `result`

Blocked-before-execution failures also include:

- `error_message`
- `failure_kind = "blocked_pre_execution"`
- `executed = false`

Reason: replay needs explicit blocked-vs-executed failure visibility without inventing new canonical event types.

#### `system.run.completed`

Must keep native-compatible counters:

- `final_output`
- `stop_reason`
- `step_count`
- `tool_call_count`
- `input_tokens`
- `output_tokens`
- `total_tokens`

Reason: scoring already reads these fields.

### Event ordering rules

These are protocol invariants, not suggestions.

#### Run start

- first accepted event must be `system.run.started`
- `system.run.started` is emitted when Python actually begins the reasoning loop, immediately before the first step/model-call sequence, not when `POST /reasoning/runs` is merely accepted

Reason: Go should not synthesize missing lifecycle events.

#### Step meaning

One step means one model turn:

- one provider call,
- plus any resulting ordered tool batch,
- plus any resulting tool events,
- then step close.

Reason: this matches native `max_iterations` semantics.

#### Tool-using turn order

Required order:

1. `system.step.started`
2. `model.call.started`
3. `model.call.completed`
4. `model.tool_calls.proposed`
5. `tool.call.*` events for the accepted batch
6. `system.step.completed`

Reason: replay should reflect causal order, and Go should execute off the proposal event only after the underlying model response has been recorded.

#### No-tool success turn order

Required order:

1. `system.step.started`
2. `model.call.started`
3. `model.call.completed`
4. `system.step.completed`
5. `system.output.finalized`
6. `system.run.completed`

Reason: the final successful step must close cleanly before run-level terminal events.

#### Failure turn order

If a turn fails after step start but before normal completion:

- emit `system.run.failed`
- do not synthesize `system.step.completed`

Reason: interrupted turns should remain interrupted in replay, matching native semantics.

### Finalization rules

- `system.output.finalized` may appear at most once
- it is emitted only after final-output validation succeeds
- after `system.output.finalized`, the only allowed next canonical event is a matching `system.run.completed`
- `system.run.completed.payload.final_output` must match the already finalized output exactly

Reason: successful traces must not contain two conflicting final answers or continued reasoning after finalization.

### Terminal freeze rules

- first accepted terminal event wins
- once execution state is terminal, exact retry of that same terminal event is harmlessly acknowledged
- any different post-terminal event is rejected

Reason: replay and scoring both assume a single terminal state for a run-agent event stream.

---

## 9. Go Workflow and Control Plane

### New execution-state row

Add `reasoning_run_executions`, keyed by `run_agent_id`, with at least:

- `run_id`
- `run_agent_id`
- `reasoning_run_id`
- `endpoint_url`
- `status`
- `deadline_at`
- `sandbox_id`
- `pending_proposal_event_id`
- `pending_proposal_payload`
- `last_event_type`
- `last_event_payload`
- `result_payload`
- `error_message`
- timestamps

Reason: the workflow needs durable mutable coordination state that `run_events` does not provide.

### Workflow shape

Use the same broad pattern as hosted execution:

1. short `StartReasoningRun` activity
2. wait on a workflow signal for actionable events
3. when proposal arrives, execute tools in Go
4. submit results back to Python
5. when terminal event arrives, finish
6. cleanup sandbox best-effort

No heartbeat long-running activity.

Reason: this fits current Temporal usage and keeps the control plane simple.

### Signal payload

The signal should carry only a minimal actionable reference, for example:

```json
{
  "event_id": "string",
  "event_type": "model.tool_calls.proposed"
}
```

Reason: the actionable payload is already durably stored on `reasoning_run_executions`.

### Actionable events

The callback path persists every accepted event to `run_events`, but only signals the workflow for:

- `model.tool_calls.proposed`
- `system.run.completed`
- `system.run.failed`

Reason: only those events change workflow control flow.

### Proposal lifecycle

Rules:

- at most one outstanding proposal at a time
- callback handler rejects a new proposal while `pending_proposal_event_id` is still set
- `pending_proposal_event_id` is cleared only after `SubmitToolResults` gets a durable success ack
- if submit fails after tool execution, keep the pending proposal intact and retry submit only

Reason: this gives the workflow one unambiguous outstanding control-plane action at a time.

### Sandbox lifecycle

Rules:

- create sandbox lazily on first accepted proposal that actually needs execution
- persist `sandbox_id`
- reattach by `sandbox_id` for later batches
- destroy by `sandbox_id` on success, failure, timeout, or cancellation
- cleanup is best-effort and does not flip an already terminal run result

Reason: one sandbox per run preserves state while keeping tool-free runs cheap.

### Retry policy

- sandbox-mutating tool-execution activities: `MaximumAttempts: 1`
- idempotent bridge activities (`StartReasoningRun`, `SubmitToolResults`, `CancelReasoningRun`): small retry budget is allowed

Reason: rerunning tool execution is unsafe, but rerunning idempotent bridge calls is acceptable.

### Protocol violation handling

If Go proves the callback stream is invalid, it should:

- mark the reasoning execution failed in control-plane state,
- transition the run-agent to failed,
- rebuild replay,
- avoid injecting synthetic canonical failure events.

Reason: deterministic protocol failures should fail fast rather than decaying into deadline timeouts.

---

## 10. Python Runtime Plan

### Runtime style

Build a first-party Python state machine for ReAct.

Do not use:

- `Agent.run()`
- `agent.run_stream()`
- tool decorators as the core runtime abstraction
- `pydantic-evals`

Reason: the reasoning runtime must pause at `model.tool_calls.proposed` and let Go own the actual tool execution boundary.

### Required Python pieces

- Pydantic models for bridge request/response validation
- `ModelClient` abstraction for OpenAI-compatible chat completions
- ReAct state machine
- local WAL / durable execution store
- callback emitter

Reason: this is the smallest substrate needed to build the lane without taking on framework lock-in.

### Provider support

Only support:

- `provider_key = "openai"`
- OpenAI-compatible `/chat/completions`
- non-streaming calls

Fail fast for anything else.

Reason: this keeps v0 at parity with the provider support the Go worker actually has today.

### Pre-run validation timing

All pre-run input guardrails must finish before the first model call starts.

Reason: a blocked input should not consume provider budget or emit runtime events before validation completes.

### Strategy

`reasoning_spec` in v0 is:

```json
{
  "strategy": "react"
}
```

Step budget comes from `runtime_profile.max_iterations`.

Reason: a second reasoning-step budget knob only creates drift risk.

### Final-answer rules

#### Main loop

- if `tool_calls` is non-empty, continue with Go tool execution
- if `tool_calls` is empty and `output_text` is non-empty, candidate final answer
- if both are empty, provider/protocol failure

#### Finish reason rules

- `tool_calls` responses must also have `finish_reason == "tool_calls"`
- no-tool final answers must have `finish_reason == "stop"`
- `length`, `content_filter`, or unknown finish reasons are terminal failures

Reason: the bridge should fail fast on contradictory provider output.

### Final-output repair retry

If final-output validation fails because of answer format:

- perform another model call as a new step
- this retry is output-only and must not request tools
- if a repair call produces `tool_calls`, fail with `protocol_error`
- rejected candidates remain internal; only the accepted answer emits `system.output.finalized`

Reason: repair is a bounded finalization pass, not a re-entry into the main tool loop.

### Final-output schema source

Final-output schema validation uses the frozen `output_schema` from the execution context.

Reason: v0 should keep one source of truth for answer-shape validation.

### Callback delivery rules

- Python is the sole producer of canonical reasoning-lane `run_events`
- nonterminal callback delivery must be unambiguous
- if Python cannot tell whether a nonterminal event POST was accepted, it must stop the run and surface a bridge/protocol failure rather than retrying that event

Reason: v0 intentionally does not implement Go-side callback dedupe.

---

## 11. Guardrails and Tool Execution

### Trust-boundary split

#### Python-side checks

- pre-run input validation
- final-output validation
- final-output retry logic

Reason: these operate on reasoning inputs/outputs before or after the Go-owned execution boundary.

#### Go-side checks

- tool allowlist
- argument schema validation
- resource-limit enforcement
- shell/network policy
- post-tool sanitization before results go back to Python

Reason: the executor boundary is the real security boundary.

### Source of truth for tool policy

Tool policy comes from:

- challenge manifest
- runtime profile
- sandbox policy

Not from duplicated `guardrail_spec.pre_tool.allowed_tools` or `max_total_calls`.

Reason: v0 should keep one operative source of truth for tool capabilities and budgets.

### Tool registry

Expose the same Go-native sandbox toolset as native execution, minus `submit`.

That means:

- `read_file`
- `write_file`
- `list_files`
- `exec` only when policy allows shell

Reason: reasoning should not grow capabilities that native execution does not have.

### Prevalidation model

Before executing a proposed batch, Go validates the entire batch first.

If validation fails:

- blocked offending calls return `blocked`
- untouched companions return `skipped`
- no sandbox side effects happen

Reason: later invalid calls should not be able to invalidate earlier side effects after execution has already started.

### Blocked vs failed semantics

- allowlist/schema/resource-limit violations are nonterminal blocked tool outcomes
- true security violations are terminal run failures

Reason: the system should stay available for ordinary model mistakes, but fail closed on actual security boundary breaches.

### Budget accounting

- executed tool calls count
- blocked specific calls count
- skipped companions from a rejected batch do not count

Those counts also drive `system.run.completed.tool_call_count`.

Reason: tool budgets and completion counters should reflect actual processed or policy-violating calls, not bookkeeping placeholders.

---

## 12. Persistence, Replay, and Scoring

### Run-event producer model

Python is the only producer of canonical reasoning-lane `run_events`.

Go does not directly persist `tool.call.*` events for this lane.

Reason: a single producer keeps append ordering coherent with the current DB model.

### Tool-event echo rule

After Go accepts a tool-results batch, Python must emit exactly one canonical non-skipped `tool.call.*` event per accepted non-skipped result:

- same order
- same `tool_call_id`
- same status
- same sanitized content

Reason: replay and scoring must describe what Go actually executed, not a Python reinterpretation of it.

### Replay changes

Scoring needs no v0 logic changes.

Replay needs one small addition:

- render `model.tool_calls.proposed` as a first-class replay step, for example "Tool calls proposed"

Reason: proposal events are a durable and meaningful handoff point in this lane.

### Event-count and failure semantics

- blocked-before-execution `tool.call.failed` events count as failures for scoring
- skipped placeholder results are not persisted as canonical events

Reason: blocked calls are still model/tool-use failures; skipped placeholders are only conversation bookkeeping.

### Canonical payload consistency rules

Go should reject a stream if:

- `model.tool_calls.proposed.tool_calls` does not exactly match the immediately preceding `model.call.completed.tool_calls` for that step
- post-finalization events appear
- `system.run.completed.final_output` does not match finalized output
- a terminal event arrives while a proposal is still pending

Reason: without these invariants, canonical traces can drift away from the actual executed conversation.

---

## 13. Failure Handling and Recovery

### Start rejection

If Python rejects the run before acceptance:

- fail the start activity
- mark control-plane state failed
- do not persist canonical runtime events

Reason: the run never actually began.

### Partial event streams

Partial streams are acceptable.

- replay already handles interrupted/open steps
- scoring already has partial evaluation modes
- Go should not synthesize canonical terminal events just to make the trace look cleaner

Reason: preserving real evidence is better than inventing fake completion.

### Crash and timeout behavior

- workflow deadline still bounds the run
- if Python stops responding, Go marks execution failed in control-plane state
- runs are not resumable in v0

Reason: bounded restart-from-scratch is enough for the first slice.

### Stop reasons

`system.run.failed.payload.stop_reason` may include reasoning-specific values such as:

- `output_validation_failed`
- `protocol_error`

Reason: v0 needs to distinguish reasoning-lane protocol and validation failures from generic provider failures.

### Cancellation

- cancellation is a control-plane request to Python
- Go performs cleanup regardless of whether Python emits anything further
- do not require a canonical `system.run.failed` for cancellation

Reason: the domain already distinguishes cancellation from ordinary failure at the run level.

### Post-terminal behavior

Once a terminal event is accepted:

- the stream is frozen
- exact retry of the same terminal event is tolerated
- any different callback is rejected

Reason: terminal state must be immutable for replay/scoring correctness.

---

## 14. Delivery Plan

### Phase 1: Contracts and Schema

- add `agent_kind = "reasoning_v1"` support in DB/API validation
- add `source = "reasoning_engine"` in `runevents`
- define Go/Python bridge models
- add `reasoning_run_executions` migration and repository methods
- define callback route and ingestion interfaces

Reason: delivery starts with durable contracts and state model changes, not Python strategy work.

### Phase 2: Go Control Plane

- implement routing branch in `RunAgentWorkflow`
- add start/submit/cancel activities
- add minimal workflow signal type for reasoning
- add pending-proposal state handling
- add lazy sandbox create/attach/destroy-by-id support
- add protocol-validation failure path

Reason: Python cannot be integrated safely until Go can own the bridge and sandbox lifecycle.

### Phase 3: Python Runtime

- implement typed start state model
- implement OpenAI-compatible `ModelClient`
- implement ReAct loop
- implement callback emitter
- implement start/tool-results/cancel idempotency on durable local state
- implement final-output validation and repair retry

Reason: once the Go control plane is fixed, Python can target a stable contract.

### Phase 4: Trace Integration

- emit canonical `system.*`, `model.*`, and `tool.*` events in required order
- add replay-builder support for `model.tool_calls.proposed`
- add contract tests for payload parity and ordering

Reason: replay correctness is part of the runtime contract, not a later polish item.

### Phase 5: Verification

- end-to-end tests for tool-free success
- end-to-end tests for tool-using success
- end-to-end tests for blocked tool batch
- end-to-end tests for protocol violations
- end-to-end tests for cancellation and timeout
- golden trace tests through the existing Go scorer

Reason: the highest-value validation for v0 is bridge correctness and trace correctness.

---

## 15. Success Criteria

v0 is done when all of the following are true:

- a build with `agent_kind = "reasoning_v1"` routes into the new lane when the global flag is enabled
- Python receives one frozen `execution_context` start payload and runs a ReAct loop
- Go remains the only executor of sandbox tools
- tool-free runs can complete without ever creating a sandbox
- tool-using runs create one sandbox lazily, reuse it across turns, and clean it up best-effort
- reasoning callbacks persist canonical `run_events` with `source = "reasoning_engine"`
- replay renders the run coherently, including `model.tool_calls.proposed`
- existing scoring works without a reasoning-specific scorer
- protocol violations fail fast in control-plane state instead of silently corrupting the stream
- contract and end-to-end tests cover the accepted v0 invariants

Anything beyond that is a post-v0 follow-up.
