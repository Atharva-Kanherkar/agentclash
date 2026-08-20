# Curated "good first issue" candidates

A ready-to-file worklist for maintainers. These are **proposals** — skim each,
confirm the pointers still hold, then file the ones you like. Each is small,
well-scoped, and has explicit acceptance criteria so a newcomer can finish it
without a back-and-forth.

File one with the GitHub CLI. The `area:*` labels are live — `area:backend`,
`area:cli`, `area:web`, `area:runtime`, `area:docs`, `area:ci`, `area:other`:

```bash
gh issue create \
  --title "Add a /healthz/ready readiness probe" \
  --label "good first issue" --label "area:backend" \
  --body-file - <<'EOF'
...acceptance criteria from below...
EOF
```

> Aim to keep **8–12 open** at any time. When the pool runs low, refresh it — see
> `docs/maintainers/growth-checklist.md`.

---

## Backend

### 1. Add a `/healthz/ready` readiness probe
**Labels:** `good first issue`, `area:backend`
**Context:** Only `/healthz` (liveness) is registered (`backend/internal/api/server.go`,
`backend/internal/api/health.go`). `/healthz/ready` already appears in the
auth-skip test (`backend/internal/api/middleware_test.go:57`) but no route serves it.
**Acceptance:**
- `GET /healthz/ready` returns `200` with a small JSON body when Postgres and
  Temporal are reachable, `503` otherwise.
- Route registered next to `/healthz` in `server.go`.
- Unit test covering ready/not-ready.
- `docs/api-server/openapi.yaml` updated.

## CLI

### 2. Add an "Examples" block to `agentclash --help`
**Labels:** `good first issue`, `area:cli`
**Context:** Top-level help lists commands but no end-to-end example.
**Acceptance:** Root command shows 2–3 copy-pasteable examples (auth → eval start → scorecard); existing CLI tests still pass.

## CI / tooling

### 3. Lint the example challenge packs in CI
**Labels:** `good first issue`, `area:ci`
**Context:** `examples/challenge-packs/*.yaml` (12 packs) aren't validated, so they
can silently drift from the schema.
**Acceptance:** A CI job (or step) runs `agentclash challenge-pack validate` (or schema
validation) over every example; fails on an invalid pack.

### 4. Add an `.editorconfig`
**Labels:** `good first issue`, `area:other`
**Context:** No `.editorconfig`, so indentation/charset varies by editor.
**Acceptance:** Root `.editorconfig` (tabs for Go and Makefiles, 2-space YAML/JSON, final newline, UTF-8); matches existing files so it produces no diff churn.

## Docs

### 5. Cross-link the zero-key dev profile from the docs site
**Labels:** `good first issue`, `area:docs`
**Context:** CONTRIBUTING now documents the "runs with zero API keys" profile; the
docs site self-host page doesn't mention it.
**Acceptance:** Self-host / getting-started docs link the zero-key profile and the tiered setup.

### 6. Add a "deeper smoke test" option to `make doctor`
**Labels:** `good first issue`, `area:backend`, `area:docs`
**Context:** `make doctor` aliases the local-stack ownership and health status.
`scripts/dev/curl-create-run.sh` can additionally exercise a real create-run
flow.
**Acceptance:** An opt-in flag/target (e.g. `make doctor DEEP=1`) runs the curl smoke test and reports pass/fail; documented.
