# Worker scaling and task-queue topology

Fleet workers process Temporal workflows and activities for eval runs. This
document covers concurrency knobs and the execution / scoring / background
queue split introduced with Fleet 2.

## Task queues

| Queue | Purpose |
|---|---|
| `execution` | Eval-session, run, run-agent, and agent-harness workflows plus native/prompt/multi-turn execution activities |
| `scoring` | `score_run_agent`, scorecard build, replay build |
| `background` | Dataset generation and public agent tryouts |

Workflows are started on `execution` (or `background` for tryouts/dataset jobs).
Activities pin to their class via `ActivityOptions.TaskQueue`.

## Environment variables

| Variable | Default | Meaning |
|---|---|---|
| `WORKER_TASK_QUEUES` | `execution,scoring,background` | Comma-separated queues this process serves |
| `WORKER_TASK_QUEUE` | _(unset)_ | Legacy single-queue override; `RunWorkflow` expands to all three classes |
| `WORKER_MAX_CONCURRENT_ACTIVITIES` | `100` | Per-queue Temporal `MaxConcurrentActivityExecutionSize` |
| `WORKER_MAX_CONCURRENT_WORKFLOW_TASKS` | `100` | Per-queue Temporal `MaxConcurrentWorkflowTaskExecutionSize` |
| `WORKER_ACTIVITIES_PER_SECOND` | `0` | Worker-local rate limit; `0` = unlimited |
| `WORKER_TASKQUEUE_ACTIVITIES_PER_SECOND` | `0` | Server-side queue rate limit; `0` = unlimited |

Bounded fan-out caps inside workflows (deterministic defaults, not env-read):

- Eval-session child runs: 16
- Run-agent children per run: 8
- Scoring activities per run: 16

## Topologies

### One-process (default / local)

```bash
export WORKER_TASK_QUEUES=execution,scoring,background
make worker
```

One binary starts three Temporal workers (one per queue) with the same identity
prefix. Capability matches the historical single-queue worker.

### Split fleet (production)

Run separate Deployments/processes:

```bash
# Execution pool (scale with run depth)
WORKER_TASK_QUEUES=execution

# Scoring pool
WORKER_TASK_QUEUES=scoring

# Background pool (dataset gen / tryouts)
WORKER_TASK_QUEUES=background
```

Background load cannot consume execution slots because the queues are distinct.
Pair with KEDA Temporal queue-depth autoscaling (Fleet 12) when ready.

## Sandbox capacity (Fleet 3)

| Variable | Default | Meaning |
|---|---|---|
| `SANDBOX_MAX_CONCURRENT` | `0` | Live sandbox cap; `0` = unlimited (today's behavior) |
| `SANDBOX_ACQUIRE_TIMEOUT` | `5m` | Max wait for a capacity slot |
| `SANDBOX_WARM_POOL_SIZE` | `0` | Per-worker warm pool size per `(template, tool_policy)` key; `0` = off |
| `SANDBOX_WARM_POOL_TTL` | `10m` | Idle warm sandbox TTL |
| `SANDBOX_PROVIDER` | `unconfigured` | `unconfigured` \| `e2b` \| `docker` |

When `REDIS_URL` is set and `SANDBOX_MAX_CONCURRENT > 0`, the budget is shared
across worker replicas (Redis ZSET + TTL). Otherwise the budget is in-process.
The warm pool is always per-worker in v1.

## Rollback / compatibility

- Setting `WORKER_TASK_QUEUE=RunWorkflow` expands to all three class queues.
- Workflow histories recorded before bounded fan-out replay via `GetVersion`
  DefaultVersion (unbounded launch-all).
- Case fan-out (`profile_config.case_fanout`) remains default-off (Fleet 1).
- Sandbox capacity / warm pool / docker provider are default-off.
