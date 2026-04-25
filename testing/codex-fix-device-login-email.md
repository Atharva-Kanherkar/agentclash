# codex/fix-device-login-email — Test Contract

## Functional Behavior
- A browser-authenticated user approving CLI device login should still end up with an email in backend auth/session responses even when the WorkOS access token omits the `email` claim.
- The backend should fetch user identity from a trustworthy WorkOS source, not from a client-supplied JSON field.
- Existing users with blank stored emails should be backfilled when WorkOS user info returns an email.
- If the WorkOS userinfo lookup fails, an otherwise valid authenticated request should still succeed for existing users, and login should not regress beyond today's behavior.
- CLI `auth login` should continue printing the best available identity, and device-login users with a recovered backend email should see the email instead of only `user_id`.

## Unit Tests
- `TestWorkOSAuthenticator_FallsBackToUserInfoEmailWhenClaimMissing` — fetches WorkOS userinfo with the bearer token, backfills the existing blank email, and returns the recovered email.
- `TestWorkOSAuthenticator_UserInfoFailureDoesNotBlockExistingUserAuth` — logs in an existing user without failing auth when the WorkOS userinfo call errors.
- `TestWorkOSUserInfoURL` — derives the correct `/oauth2/userinfo` endpoint from both root issuers and issuers that include a `/user_management/...` path.

## Integration / Functional Tests
- `cli/cmd` login fallback tests remain green, proving the CLI still prints email when `/v1/auth/session` includes it.
- Protected device-approval auth continues to use the normal WorkOS-authenticated backend path with no public API trust change.

## Smoke Tests
- `cd backend && go test ./internal/api`
- `cd cli && go test ./cmd ./internal/auth`

## E2E Tests
- N/A — not applicable for this change.

## Manual / cURL Tests
- Log in on staging with a WorkOS account whose backend user row has a blank email and run `agentclash auth login`; verify the success output shows the email instead of only the `user_id`.
- Run `agentclash auth status` after the device login and verify the `Email` field is populated.
