# Codex AgentClash Auth Login — Test Contract

## Functional Behavior
- `/auth/login` must present an AgentClash-owned login experience and must not expose "Sign in with WorkOS" as the primary call to action.
- The login action must keep using the existing WorkOS AuthKit hosted authorization redirect through `getSignInUrl({ returnTo })` so password reset, email verification, SSO routing, MFA, bot protection, and session handling remain delegated to AuthKit.
- Existing `returnTo` behavior must be preserved: unsafe values and unsupported paths fall back to `/dashboard`, and `/auth/device?user_code=...` remains supported for CLI device login.
- The PR must document the WorkOS research outcome: hosted AuthKit can be branded on the free AuthKit tier, the default hosted domain remains `*.authkit.app`, custom AuthKit domains are a paid add-on, and fully headless custom UI is possible but broader than this safe branding PR.
- No backend auth, token validation, CLI device login, or WorkOS callback semantics should change.

## Unit Tests
- `web/src/lib/auth/__tests__/return-to.test.ts` must continue to pass unchanged or with equivalent coverage.
- Add or update focused tests for login UI rendering so the primary action says `Continue with AgentClash` and does not include `Sign in with WorkOS`.
- Add or update focused tests for `signInAction` if implementation behavior changes; otherwise keep existing server action behavior intact.

## Integration / Functional Tests
- Run the relevant web test suite for auth/login and return-to behavior.
- Run lint for touched frontend files where practical.

## Smoke Tests
- Start the Next.js app locally if practical and load `/auth/login` to confirm the page renders.
- Confirm the page includes AgentClash branding and the login form submits to the existing server action.

## E2E Tests
- N/A — a full WorkOS hosted login round trip requires configured WorkOS credentials and an external identity flow.

## Manual / cURL Tests
```bash
cd web
npm run test -- src/lib/auth/__tests__/return-to.test.ts src/app/auth/login
# Expected: tests pass.

npm run lint -- src/app/auth/login src/lib/auth
# Expected: lint passes for touched auth/login files.
```
