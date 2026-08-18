# codex/issue-1235-landing-scoreboard — Test Contract

## Functional Behavior

- The logged-out homepage selects the newest real benchmark report from the
  server-side benchmark registry. Reports marked `sample: true` are never
  promoted.
- Only plain, JSON-serializable fields needed by the landing section cross the
  Server Component to Client Component boundary: slug, title, verdict,
  challenge-pack label, featured model, and result rows.
- When a real report with result rows exists, the first section after the hero
  identifies it as measured proof, renders the existing `BenchmarkScoreboard`,
  and links to both `/benchmarks/{slug}` and `/benchmarks`.
- When no real report exists, or the selected report has no result rows, the
  benchmark proof section renders nothing and the rest of the homepage remains
  unchanged.
- The scoreboard markup remains centralized in
  `web/src/components/marketing/benchmark-scoreboard.tsx`.

## Unit Tests

- Homepage benchmark selection chooses the first non-sample report from the
  already date-sorted registry.
- Homepage benchmark selection returns no value when every report is a sample.
- Homepage benchmark selection returns no value when the real report has no
  result rows.
- The proof section renders report copy, model rows, measured scores, and both
  expected links.
- The proof section renders no markup when benchmark data is absent.

## Integration / Functional Tests

- `web/src/app/page.tsx` reads benchmark data on the server and passes the
  selected narrow object into `HomePage`.
- `HomePage` places the proof section immediately after the hero and before the
  Replay feature section.
- Existing benchmark scoreboard tests remain green.

## Smoke Tests

- `cd web && pnpm exec vitest run` passes.
- `cd web && pnpm lint` passes.
- `cd web && pnpm exec tsc --noEmit` passes.
- `cd web && pnpm build` passes.

## E2E Tests

N/A — this change promotes repository-backed static benchmark content and does
not introduce a new interactive workflow.

## Manual / cURL Tests

- Render the logged-out homepage at 375px and 1440px and confirm the scoreboard
  remains readable through horizontal scrolling on narrow screens.
- Confirm "Reproduce this run" opens the selected benchmark detail route and
  "Browse all benchmarks" opens the benchmark hub.
- Confirm the section is absent when all benchmark reports are temporarily
  treated as samples in a test fixture; no production content is modified for
  this check.
