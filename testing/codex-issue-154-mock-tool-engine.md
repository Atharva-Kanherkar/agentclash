# codex/issue-154-mock-tool-engine — Test Contract

## Functional Behavior
- Replace the placeholder `manifestBackedTool` mock execution (which returns "not implemented yet") with a real mock tool engine that returns deterministic, controlled responses.
- Implement three mock strategies:
  - **Keyed Lookup**: match a specific parameter value (`lookup_key`) against a `responses` map; `"*"` is the wildcard/fallback; missing key with no fallback returns structured error.
  - **Static**: return the same `response` regardless of input arguments.
  - **Echo/Template**: return a `template` response with `${param_name}` placeholders replaced by the actual argument values.
- Mock tools must:
  - Implement the existing `Tool` interface (`Name`, `Description`, `Parameters`, `Category`, `Execute`).
  - Return `ToolCategoryMock` from `Category()`.
  - Never call any `sandbox.Session` methods — execution is pure data, no sandbox interaction.
  - Return structured JSON content on success (not `IsError`).
  - Return structured error JSON when lookup fails without fallback.
  - Be indistinguishable from real tools in the agent's view (same `ToolDefinition` shape).
- Run events must record mock tool calls through the existing `Observer.OnToolExecution` path with `ToolCategory = "mock"`.
- Mock tool configuration is validated at tool creation time (`newManifestCustomTool`): invalid strategy, missing required fields, or malformed template vars should return an error during registry build, not at execution time.
- The `implementation` JSON shape for mock tools:
  - `type: "mock"` (required, already detected)
  - `strategy: "lookup" | "static" | "echo"` (required; defaults to "static" if only `response` is present)
  - Strategy-specific fields: `lookup_key` + `responses` for lookup; `response` for static; `template` for echo.

## Unit Tests
- `TestMockTool_StaticStrategy` — returns the configured static response regardless of input args.
- `TestMockTool_LookupStrategy_MatchesKey` — returns the response for the matching key value.
- `TestMockTool_LookupStrategy_FallbackWildcard` — returns the `"*"` response when the key doesn't match any explicit entry.
- `TestMockTool_LookupStrategy_NoMatchNoFallback` — returns structured error when key has no match and no `"*"` fallback.
- `TestMockTool_EchoStrategy_SubstitutesParameters` — replaces `${param}` placeholders with actual argument values.
- `TestMockTool_EchoStrategy_MissingParamLeavesPlaceholder` — `${missing}` placeholder left as-is when param not provided.
- `TestMockTool_CategoryIsMock` — tool returns `ToolCategoryMock`.
- `TestMockTool_ZeroSandboxInteraction` — executing a mock tool with a nil session does not panic.
- `TestNewManifestCustomTool_RejectsInvalidMockStrategy` — returns error for unknown strategy.
- `TestNewManifestCustomTool_RejectsLookupWithoutKey` — returns error when lookup strategy has no `lookup_key`.
- `TestBuildToolRegistry_MockToolsVisibleAndExecutable` — mock tools appear in visible set and execute successfully.
- `TestMockTool_LookupStrategy_NestedKeyValue` — lookup key extracts from nested argument correctly (string match only).

## Integration / Functional Tests
- Native executor integration: mock tools resolve from the registry and execute during a tool call loop, returning structured content to the LLM provider.
- Observer records mock tool executions with `ToolCategory = "mock"` in the event payload.

## Smoke Tests
- `go test ./internal/engine/... ./internal/sandbox/...` — all existing tests continue to pass.
- New mock tool tests pass.

## E2E Tests
- N/A — this is backend engine behavior, not a user-facing browser flow.

## Manual / cURL Tests
```bash
cd /Users/atharva/agentclash/backend
go test ./internal/engine/... -run 'TestMockTool|TestBuildToolRegistry_Mock|TestNewManifestCustomTool'

# Expected: all listed tests pass
# Existing tests remain green:
go test ./internal/engine/... ./internal/sandbox/...
```
