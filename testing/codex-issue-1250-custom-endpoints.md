# codex/issue-1250-custom-endpoints — Test Contract

Issue: #1250 — Custom OpenAI-compatible endpoints

## Functional Behavior

- Persist `provider_accounts.base_url` with a non-null empty-string default and round-trip it through repository rows, REST responses, dataset account lookups, and worker execution context generation.
- Canonicalize provider keys and accept exactly `openai`, `anthropic`, `gemini`, `xai`, `openrouter`, `mistral`, and `custom`; reject unknown providers.
- Accept an optional endpoint override for supported providers and require one for `custom`. Normalize accepted URLs without changing their origin or path semantics.
- Before writing an account or credential secret, require endpoint overrides to be absolute HTTPS URLs without userinfo, query, or fragment components and whose complete DNS result set contains only public, globally routable addresses.
- Reject private, loopback, link-local, metadata, carrier-grade NAT, multicast, unspecified, documentation, benchmarking, reserved, and other special-use addresses, including a hostname with a mixed public/blocked DNS answer set.
- Revalidate endpoint overrides when dialing, connect only to the validated addresses, and reject redirects outside the originally validated origin.
- Add request-level endpoint overrides to provider invocation and model-listing contracts. A request override takes precedence over adapter defaults, while `custom` without an override fails locally without contacting OpenAI.
- Route `custom` through the OpenAI-compatible adapter and forward account endpoints through connection tests/model discovery, native and prompt executors, simulators, matching deployment judges, dataset-generation roles, and ranking insights.
- Register `custom` in local provider, tryout, smoke-model, pricing, and credential-reference registries. Its environment variable is `CUSTOM_API_KEY`, it has no default smoke model, and its static pricing remains unknown.
- Keep custom models invalid as pack judges when no explicit supported provider can be inferred. OpenAI Responses/research mode and per-account vault naming/throttle changes are outside this change.
- Let users create a “Custom / OpenAI-compatible” account with a conditionally visible required endpoint URL, test accounts with a required custom model (optional for providers with defaults), and inspect pass/fail message, provider model, and duration.
- Use a shared provider-account label throughout selectors and display only the endpoint host (never credentials or URL path/query data) in labels and the provider table.
- Document the provider enum, `base_url`, and provider-account test request/response in OpenAPI.

## Unit Tests

- API/provider-key tests cover trimming/case canonicalization, allowlisting, required custom URL, normalization, malformed/insecure URLs, disallowed URL components, blocked literals, blocked/mixed DNS results, and validation occurring before repository or secret writes.
- Provider endpoint-safety tests cover public address acceptance, special-use address rejection, DNS changes between validation and use, dialing only a validated IP, redirect same-origin acceptance, and cross-origin redirect rejection.
- Provider adapter tests cover invocation/model-list override precedence, missing custom URL, custom routing, and no accidental request to the OpenAI default.
- Repository and execution-context tests cover `base_url` scan/round-trip behavior.
- Capturing-client tests cover endpoint forwarding from every account-backed invocation path listed under Functional Behavior.
- Registry tests cover local provider support, `CUSTOM_API_KEY`, no custom smoke default, unknown pricing, tryout support, and preserved custom pack-judge rejection.
- Web tests cover conditional create fields and payloads, endpoint-host labels, custom model requirements, provider-default model behavior, smoke-test request payloads, and passed/failed result rendering.

## Integration / Functional Tests

- `make check-backend` passes.
- `make check-runtime` passes.
- `make check-web` passes.
- OpenAPI validation/generation checks included by repository gates pass with the new schemas and route.

## Smoke Tests

- Against a controlled public HTTPS OpenAI-compatible server, create a custom account, list models through the account endpoint, and run the provider-account test with an explicit model.
- Confirm the server observes `/models` and `/chat/completions` requests under the configured base path and receives the configured bearer credential.
- Confirm a custom account without a URL and an endpoint resolving to blocked space fail before any outbound provider request.

## E2E Tests

- Provider-account page flow: select Custom, enter name/credential/HTTPS endpoint, create, observe host-only table label, open Test, enter model, and observe a passed result.
- Failure flow: test with an invalid model or controlled provider failure and observe a failed result with message, model, and duration rather than a dialog transport error.

## Manual / cURL Tests

With `$API_URL`, `$TOKEN`, `$WORKSPACE_ID`, and a controlled `$COMPAT_BASE_URL` configured:

```bash
curl -sS -X POST "$API_URL/v1/workspaces/$WORKSPACE_ID/provider-accounts" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"provider_key\":\"custom\",\"name\":\"controlled-compatible\",\"api_key\":\"test-key\",\"base_url\":\"$COMPAT_BASE_URL\"}"

curl -sS "$API_URL/v1/provider-accounts/$ACCOUNT_ID/models" \
  -H "Authorization: Bearer $TOKEN"

curl -sS -X POST "$API_URL/v1/provider-accounts/$ACCOUNT_ID/test" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"controlled-model"}'
```

Also submit an HTTP URL, a URL with userinfo/query/fragment, a loopback/metadata URL, and a hostname returning mixed public/private answers; each must return a validation response and leave no account or secret behind.
