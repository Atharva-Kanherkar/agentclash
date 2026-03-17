# AgentClash Design System

Established from the landing page (landing-2). These tokens and patterns apply to every page — landing, dashboard, replay viewer, scorecards, comparison, settings.

## Aesthetic Direction

**Editorial precision meets dev tool.** Serif headlines for page titles, monospace for data, sans-serif for everything else. Warm palette. Centered layouts for marketing, left-aligned data for dashboards. The product should feel like a well-made instrument, not a flashy SaaS template.

Inspirations: Resend (editorial warmth), Anthropic (terracotta accent), Linear (hierarchical opacity), Stripe (confidence through restraint).

---

## Typography

Three font families. Each has a specific role.

### Display: Instrument Serif

- **Where:** Page titles, section headings, hero text
- **Weight:** 400 (regular only — this font is elegant at normal weight)
- **CSS variable:** `--font-display`
- **Google Fonts import:** `Instrument_Serif`, weight 400

### Body: DM Sans

- **Where:** Paragraphs, buttons, labels, navigation, form fields, descriptions
- **Weight range:** 400 (body), 500 (emphasis), 600 (strong labels)
- **CSS variable:** `--font-body`
- **Google Fonts import:** `DM_Sans`

### Mono: JetBrains Mono

- **Where:** Data tables, scorecard numbers, comparison deltas, replay events, code, terminal output, metric values
- **Weight range:** 400 (data), 500 (column headers), 600 (highlighted values like deltas)
- **CSS variable:** `--font-mono`
- **Google Fonts import:** `JetBrains_Mono`, weights 400/500/600

### Usage rules

- **Never** use the display font for body text or data
- **Never** use the body font for numeric data in tables — always mono
- **Never** use mono for marketing copy or descriptions
- Page titles: display font
- Section labels (uppercase small): body font, weight 600
- Data tables: mono font throughout
- Buttons: body font
- Navigation: body font

---

## Color

Two themes: dark (default) and light. Both use warm tones — no pure black, no pure white.

### Dark theme

```css
--bg: #14120F;               /* warm near-black, not #000 */
--bg-alt: #1A1815;            /* alternate section background */
--border: rgba(237,233,225,0.06);
--text-1: #EDE9E1;            /* primary text — warm off-white */
--text-2: rgba(237,233,225,0.55);  /* secondary — descriptions, body */
--text-3: rgba(237,233,225,0.28);  /* tertiary — labels, muted */
--text-4: rgba(237,233,225,0.12);  /* quaternary — section headers, dividers */
--accent: #D97757;            /* terracotta — CTAs, warnings, emphasis */
--surface: rgba(237,233,225,0.03); /* card/panel backgrounds */
```

### Light theme

```css
--bg: #FAF8F5;               /* warm off-white, not #fff */
--bg-alt: #F0EDE8;            /* alternate section background */
--border: rgba(20,18,15,0.08);
--text-1: #1A1815;            /* primary text — warm near-black */
--text-2: rgba(26,24,21,0.55);
--text-3: rgba(26,24,21,0.28);
--text-4: rgba(26,24,21,0.1);
--accent: #C4593C;            /* darker terracotta for light backgrounds */
--surface: rgba(20,18,15,0.025);
```

### Text hierarchy

Use the four text levels consistently:

| Level | Token | When to use |
|---|---|---|
| `--text-1` | Primary | Headlines, important data, active states |
| `--text-2` | Secondary | Body text, descriptions, most readable content |
| `--text-3` | Tertiary | Labels, navigation items, muted information |
| `--text-4` | Quaternary | Section headers (uppercase), divider labels, disabled states |

**Rule:** Never hard-code gray hex values. Always use the opacity-based tokens. They adapt to both themes automatically.

### Accent usage

Terracotta (`--accent`) is used sparingly:

- Highlighted values (regression deltas, warnings)
- CTAs on the landing page (but dashboards use `--text-1` for primary buttons)
- The "clash" in "agentclash" logo
- Release gate "WARN" badges
- Emphasized words in prose ("_actually_ better")

**Do not** use accent for success states. Success is communicated through context (pass verdicts, completed statuses), not a green color. If a green is needed for explicit pass/fail badges, use `#22C55E` sparingly.

### Semantic colors (dashboards only)

For dashboard UI that needs explicit pass/fail/warning states:

```css
--status-pass: #22C55E;
--status-fail: #EF4444;
--status-warn: #D97757;       /* same as accent */
--status-pending: rgba(237,233,225,0.28); /* same as text-3 */
```

---

## Spacing

No rigid spacing scale. Instead, follow these principles:

- **Sections:** 64px vertical padding between major sections
- **Within sections:** 24-36px between title and content
- **Content max-width:** 640px for reading content, 960px for nav/wide layouts
- **Padding:** 32px horizontal padding on content containers
- **Table cells:** 7-10px vertical, 16px horizontal
- **Buttons:** 12px vertical, 28px horizontal

---

## Borders and Surfaces

- **Border color:** `var(--border)` — very subtle, just enough to separate
- **Border radius:** 6px for buttons, 8-10px for panels/tables, 4px for badges
- **Surface background:** `var(--surface)` for table headers, panel backgrounds
- **Alternate background:** `var(--bg-alt)` for alternating page sections

---

## Data Table Pattern

This is the most important pattern — it appears in scorecards, comparisons, replays, and run details.

### Structure

```
┌─ Header bar (surface bg, 10px 16px padding) ──────────────────┐
│  [window dots]  title                           context label  │
├─ Column headers (10px font, uppercase, text-4) ───────────────┤
│                              base      cand      delta         │
├─ Data rows (mono font, 13px, 7px 16px padding) ──────────────┤
│  correctness                1.0000    1.0000      0.00         │
│  reliability                1.0000    0.8000     −0.20  ←accent│
│  tokens                       847     1,178      +331          │
├─ Footer bar (surface bg) ─────────────────────────────────────┤
│  label                                    VERDICT VALUE        │
└───────────────────────────────────────────────────────────────┘
```

### Rules

- Entire table uses `--font-mono`
- Numbers are right-aligned
- Labels are left-aligned, lowercase
- Column headers are uppercase, 10-11px, `--text-4`
- Header bar has three dot circles (window chrome) + title
- Footer bar has verdict/summary
- Negative deltas in `--accent` color
- Positive/neutral deltas in `--text-2` or `--text-3`
- Border between rows: `1px solid var(--border)`
- Border radius on outer container: 10px

---

## Section Label Pattern

Used throughout landing and dashboard pages to introduce sections:

```
WHAT AGENTCLASH DOES          ← 11px, font-weight 600
                                 letter-spacing 0.14em
                                 text-transform uppercase
                                 color: var(--text-4)
                                 margin-bottom: 24px
```

---

## Button Patterns

### Primary (landing page)

```css
background: var(--text-1);
color: var(--bg);
border-radius: 6px;
font-size: 14px;
font-weight: 500;
padding: 12px 28px;
```

### Primary (dashboard)

Same as landing page primary.

### Secondary / Ghost

```css
background: transparent;
border: 1px solid var(--border);
color: var(--text-2);
border-radius: 6px;
font-size: 13px;
padding: 8px 20px;
```

### Accent (rare — warnings, destructive)

```css
background: var(--accent);
color: var(--bg);
```

---

## Badge Pattern

For status indicators (run status, scorecard verdict, replay state):

```css
font-family: var(--font-mono);
font-size: 11px;
font-weight: 600;
text-transform: uppercase;
letter-spacing: 0.06em;
padding: 4px 10px;
border-radius: 4px;
```

Variants:
- **Pass:** `color: #22C55E; background: rgba(34,197,94,0.1)`
- **Fail:** `color: #EF4444; background: rgba(239,68,68,0.1)`
- **Warn:** `color: var(--accent); background: rgba(217,119,87,0.1)`
- **Pending:** `color: var(--text-3); background: var(--surface)`

---

## Theme Toggle

Bottom-right fixed circle button, 44px, backdrop-blur, toggles `data-theme` attribute on `<html>`.

---

## What This Does NOT Cover

- Component library / React component API (build as needed)
- Animation specifications (add when dashboard pages ship)
- Responsive breakpoints (establish during dashboard build)
- Icon set (choose when needed — Lucide is a good default)
- Form input styling (establish when settings/creation pages are built)
