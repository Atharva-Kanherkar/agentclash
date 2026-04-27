# codex/fix-nourico-icon-regression - Test Contract

## Functional Behavior
- Restore differentiated semantic icons across the web app.
- Do not map unrelated icon names to a single Nourico glyph.
- Use `lucide-react` for semantic UI icons such as GitHub, arrows, loaders, checks, alerts, navigation, actions, and status glyphs.
- Keep the Nourico vector only for places that actually mean the Nourico mark. If there are no such call sites, remove the local Nourico adapter.
- Restore `lucide-react` as a direct web dependency because app source imports it directly.
- Preserve existing app/test behavior and do not touch unrelated untracked files.

## Unit Tests
- Existing web unit tests should pass.

## Integration / Functional Tests
- `npm run lint` from `web/` should pass.
- `npm run build` from `web/` should pass.
- `npm run test -- --run` from `web/` should pass.
- `rg "@/components/ui/nourico-icons" web/src` should return no matches unless a true Nourico mark call site is intentionally present.
- `rg "from \"lucide-react\"|from 'lucide-react'" web/src` should show semantic icon imports restored.

## Smoke Tests
- Production build should render all app routes without icon module errors.

## E2E Tests
- N/A - no browser E2E suite is defined for this repair.

## Manual Tests
```bash
cd web
npm run lint
npm run build
npm run test -- --run
rg "@/components/ui/nourico-icons" src
rg "from \"lucide-react\"|from 'lucide-react'" src
```

