---
name: frontend-design
description: Create distinctive, production-grade frontend interfaces with high design quality. Use this skill when the user asks to build web components, pages, or applications. Generates creative, polished code that avoids generic AI aesthetics.
license: Complete terms in LICENSE.txt
---

This skill guides creation of distinctive, production-grade frontend interfaces that avoid generic "AI slop" aesthetics. Implement real working code with exceptional attention to aesthetic details and creative choices.

The user provides frontend requirements: a component, page, application, or interface to build. They may include context about the purpose, audience, or technical constraints.

## Design Thinking

Before coding, understand the context and commit to a BOLD aesthetic direction:
- **Purpose**: What problem does this interface solve? Who uses it?
- **Tone**: Pick an extreme: luxury/refined, editorial/magazine, soft/pastel, art deco/geometric, organic/natural, playful/toy-like, retro-futuristic, etc. There are so many flavors to choose from. Use these for inspiration but design one that is true to the aesthetic direction.
- **Constraints**: Technical requirements (framework, performance, accessibility).
- **Differentiation**: What makes this UNFORGETTABLE? What's the one thing someone will remember?

**CRITICAL**: Choose a clear conceptual direction and execute it with precision. Bold maximalism and refined minimalism both work - the key is intentionality, not intensity.

Then implement working code (HTML/CSS/JS, React, Vue, etc.) that is:
- Production-grade and functional
- Visually striking and memorable
- Cohesive with a clear aesthetic point-of-view
- Meticulously refined in every detail

## Frontend Aesthetics Guidelines

Focus on:
- **Typography**: Choose fonts that are beautiful, unique, and interesting. Pair a distinctive display font with a refined body font. Use Google Fonts for variety. Consider serif display fonts for headlines (like Domaine, Playfair Display, Instrument Serif) paired with clean sans-serif body text. Variable fonts with full weight ranges are preferred.
- **Color & Theme**: Use warm dark mode (not pure #000 — use #111, #141416, or #14120b with warm undertones). Warm off-white text (not pure #fff). Distinctive accent colors that are NOT blue or purple — consider amber, terracotta, racing red, warm cream. Use `clamp()` for responsive sizing. Use hierarchical text opacity (primary, secondary, tertiary, quaternary) instead of hard-coded grays.
- **Motion**: Focus on high-impact moments: staggered entrance animations on page load, scroll-triggered reveals, spring-based physics. ALWAYS respect `prefers-reduced-motion`. Use CSS `animation-delay` for staggered effects. Ambient background motion (subtle grid animations, gradient shifts) adds life.
- **Spatial Composition**: Centered hero with visual element below (not side-by-side). Bento grids with mixed card sizes for features. Generous whitespace. Asymmetric layouts. Step-by-step sections for processes.
- **Backgrounds & Visual Details**: Create atmosphere with gradient meshes, noise textures, warm-toned overlays. Subtle shadows (0 2px 8px rgba(0,0,0,0.08)). Glass/metal-inspired button aesthetics. Real depth through layering, not flat design.

## NEVER DO THESE

These aesthetics are explicitly banned. They look generic, dated, or like AI-generated slop:

### Banned: "Neon Gaming / Esports Arena"
- Vibrant gradient orbs floating on dark backgrounds
- Emoji icons as feature bullets (⚔️ 🎯 🏆)
- "Esports scoreboard" mockup layouts
- Gradient text (`bg-clip-text text-transparent`) as the primary visual trick
- Multiple neon glow colors competing for attention
- Round cards with gradient borders on dark backgrounds
- The word "arena" styled like a gaming tournament

### Banned: "Terminal / Hacker Aesthetic"
- All-monospace-everything pages
- Green-on-black color schemes
- Fake CLI output as the hero section
- `$` prompt symbols as design elements
- Man-page or --help flag styling
- Pages that look like a formatted README
- "Install via npm" as the primary CTA styling

### Banned: Generic AI Patterns
- Overused font families (Inter alone, Roboto, Arial, system fonts without a display pair)
- Purple gradients on white backgrounds
- Pure black (#000000) backgrounds — use warm darks
- Pure white (#ffffff) text — use warm off-whites
- Side-by-side hero layouts (text left, image right) — the 2020 SaaS template
- Stock illustrations and abstract blob shapes
- Cookie-cutter pricing tables on landing pages
- Auto-pulled Twitter/X testimonial embeds
- Flat, zero-motion pages with no animation
- The same "Get started" CTA on every button
- Blue-only accent color palettes

## What Premium Dev Tool Sites Actually Do (2025-2026 Research)

Based on analysis of Linear, Vercel, Stripe, Resend, Cursor, Anthropic, and Cal.com:

1. **Warm dark > cold dark.** Cursor uses #14120b (warm charcoal), not #000. Resend uses "Iron" and "Stone" tones. Pure black looks dated.

2. **Serif display fonts for headlines** signal sophistication. Resend uses Domaine for editorial impact. Pair with clean sans-serif body text.

3. **Custom or distinctive display fonts** are table stakes. Cursor has CursorGothic. Cal has Cal Sans. Even a bold Google Font choice beats system fonts.

4. **Non-blue accent colors** differentiate. Anthropic uses terracotta (#d97757). Consider warm amber, racing red, or sage green.

5. **Centered hero, NOT side-by-side.** Headline centered, visual element below. Two CTAs: primary bold + secondary light. Small eyebrow text above.

6. **Bento grids** for feature sections. Mixed card sizes. Asymmetric layouts. Not uniform 3-column grids.

7. **Staggered entrance animations** on scroll. Spring-based physics over linear easing. One well-orchestrated page load > scattered micro-interactions.

8. **Hierarchical text opacity** — 4+ levels of text color through reduced opacity, not hard-coded gray values.

9. **Trust signals immediately after hero** — logo carousel, GitHub stars, or usage numbers.

10. **Final CTA in a visually distinct block** with different background color.

## Implementation Quality

- Use CSS custom properties (`var()`) for all theme colors
- Use `clamp()` for responsive typography and spacing
- Use `@media (prefers-reduced-motion: reduce)` for all animations
- Use proper semantic HTML
- Test at mobile, tablet, and desktop breakpoints
- Ensure sufficient color contrast for accessibility
- Use `will-change` for GPU-accelerated animations
- Prefer `rgba()` with opacity for text hierarchy over hard-coded hex grays

Remember: Claude is capable of extraordinary creative work. Don't hold back, show what can truly be created when thinking outside the box and committing fully to a distinctive vision.
