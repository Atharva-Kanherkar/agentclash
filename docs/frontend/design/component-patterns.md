# Component Patterns

Reusable UI patterns that appear across multiple pages. Not a component library — just the patterns to follow when building components.

## Data Table

The primary data display pattern. Used in: scorecards, comparisons, run agent lists, replay step lists.

### Anatomy

```
┌─ Header ──────────────────────────────────────────────────────┐
│  [●●●]  Title                                  Context label  │
├─ Column headers ──────────────────────────────────────────────┤
│  (uppercase, mono, 10-11px, text-4)                           │
├─ Rows ────────────────────────────────────────────────────────┤
│  label          value        value        delta               │
│  label          value        value        delta               │
├─ Footer ──────────────────────────────────────────────────────┤
│  summary label                            verdict/status      │
└───────────────────────────────────────────────────────────────┘
```

### Rules

- Font: `--font-mono` for the entire table body
- Numbers: right-aligned, `font-variant-numeric: tabular-nums`
- Labels: left-aligned, lowercase
- Header/footer: `var(--surface)` background
- Row borders: `1px solid var(--border)`
- Container: `border-radius: 10px`, `overflow: hidden`
- Window dots in header: three 8px circles in `--text-4`

---

## State Banner

Shows the state of an async resource (replay, scorecard, comparison). Three states:

### Ready

Content is available. Show the data directly. No banner needed.

### Pending

Resource is still being generated.

- **HTTP status:** 202
- **Display:** Muted message explaining what's happening
- **Text color:** `--text-3`

### Errored

Resource generation failed.

- **HTTP status:** 409
- **Display:** Error message with explanation
- **Accent color for the error message**

---

## Status Badge

Compact inline indicator. Used for: run status, agent status, verdict, scoring state.

### Variants

| State | Text color | Background |
|---|---|---|
| Pass / Completed | `#22C55E` | `rgba(34,197,94,0.1)` |
| Fail / Failed | `#EF4444` | `rgba(239,68,68,0.1)` |
| Warn | `var(--accent)` | `rgba(217,119,87,0.1)` |
| Pending / Running | `var(--text-3)` | `var(--surface)` |

### Style

```css
font-family: var(--font-mono);
font-size: 11px;
font-weight: 600;
text-transform: uppercase;
letter-spacing: 0.06em;
padding: 4px 10px;
border-radius: 4px;
```

---

## Section Header

Uppercase label that introduces a content section.

```css
font-family: var(--font-body);
font-size: 11px;
font-weight: 600;
letter-spacing: 0.14em;
text-transform: uppercase;
color: var(--text-4);
margin-bottom: 24px;
```

---

## Page Title

Uses the display serif font.

```css
font-family: var(--font-display);
font-size: clamp(1.6rem, 3.5vw, 2.4rem);
line-height: 1.1;
color: var(--text-1);
```

---

## Eyebrow Label

Small accent-colored label above a page title.

```css
font-size: 12px;
font-weight: 500;
letter-spacing: 0.12em;
text-transform: uppercase;
color: var(--accent);
```

With a decorative line:
```
── Agent evaluation
```
The line is 28px wide, 1.5px tall, `var(--accent)` color.

---

## Replay Step Item

A single step in the replay timeline.

### Information shown

- Step number (position in list)
- Headline ("Model call to gpt-4.1", "Tool call: submit")
- Type badge (system / provider / tool / scoring)
- Status badge (completed / running / failed / interrupted)
- Timestamp
- Duration (if completed_at exists)
- Expandable details: provider key, model ID, tool name, error message, artifact IDs

### Layout

Vertical list, each item separated by `var(--border)`. Step number on the left, headline and badges on the right. Details expand on click.

---

## Scorecard Dimension Row

A single dimension in a scorecard view.

### Information shown

- Dimension name (correctness, reliability, latency, cost)
- Score value (numeric or "—" if nil)
- State badge (available / unavailable / error)
- Reason text (when unavailable or error)

### Layout

Horizontal row matching the data table pattern. Score in mono font, right-aligned.

---

## Comparison Delta

A single dimension delta in a comparison view.

### Information shown

- Dimension name
- Baseline value
- Candidate value
- Delta with direction indicator
- Better direction (higher/lower)
- State (available / unavailable / error)

### Layout

Matches the data table pattern from the landing page. Delta column uses accent color when the value indicates a regression.

---

## Empty State

When a page has no data yet.

- Centered text
- `--text-3` color
- Brief explanation of what will appear here
- Optional CTA to create the first resource

---

## Loading State

When data is being fetched.

- Same layout as the populated state but with placeholder blocks
- Use `var(--surface)` background for placeholder blocks
- Subtle pulse animation (CSS only)

---

## What This Does NOT Cover

- Form inputs (text fields, selects, checkboxes) — define when building creation pages
- Navigation / sidebar layout — define when building the app shell
- Modal / dialog patterns — define when needed
- Toast / notification patterns — define when needed
- Chart / visualization patterns — define when comparison or analytics pages are built
