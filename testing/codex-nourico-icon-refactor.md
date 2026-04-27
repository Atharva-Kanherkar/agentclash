# codex/nourico-icon-refactor - Test Contract

## Functional Behavior
- Every app import that previously consumed `lucide-react` should instead consume the local Nourico vector icon adapter.
- The local adapter should render the Nourico SVG path from `https://framer.com/m/Nourico-EmNcpg.js@96vlh0u4IGtzvstHLNV1`.
- Existing icon call sites should keep working with `className`, `style`, `aria-*`, `data-*`, `ref`, and standard SVG props.
- Existing type-only icon references such as `LucideIcon` should remain source-compatible.
- Spinner usages that attach `animate-spin` to `Loader2`-style icons should still animate the rendered SVG.
- Non-lucide provider/logo icons from `@lobehub/icons` are out of scope unless they are explicitly part of a local lucide replacement path.

## Unit Tests
- N/A - this is a visual icon adapter refactor without existing dedicated icon unit tests.

## Integration / Functional Tests
- `npm run lint` from `web/` should pass.
- `npm run build` from `web/` should pass.
- `rg "from \"lucide-react\"|from 'lucide-react'" web/src` should return no source imports.

## Smoke Tests
- Start the web app and verify at least one page renders without module-resolution errors.
- Verify interactive components that use icons, such as menus, dialogs, and table controls, do not crash at render time.

## E2E Tests
- N/A - no browser E2E suite is defined for this icon-only refactor.

## Manual Tests
```bash
cd web
npm run lint
npm run build
rg "from \"lucide-react\"|from 'lucide-react'" src
```

