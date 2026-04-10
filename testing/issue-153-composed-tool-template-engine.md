# issue-153-composed-tool-template-engine - Test Contract

## Functional Behavior
- `buildToolRegistry()` accepts a pre-resolved `map[string]string` secrets input for composed-tool construction.
- Secrets placeholders of the form `${secrets.KEY}` resolve at registry-build time into the composed tool's stored args template.
- Parameter placeholders resolve only at execution time from agent-provided args.
- Supported placeholder forms are `${param_name}`, `${secrets.KEY}`, `${param.nested.field}`, and `${parameters}`.
- Template resolution is single-pass. Agent-provided values containing `${...}` are treated as literals and are not recursively resolved.
- Shared template helpers are extracted from `mock_tools.go` into `engine/template.go`, and both mock tools and composed tools use them.
- When a composed tool references a missing primitive, the tool is soft-disabled from the visible registry instead of failing the whole registry build.
- Missing paths in dot-notation resolution return a composed-tool-scoped execution error.
- Array indexing is not supported in v1.
- When a template value is exactly `${parameters}`, the resolved output is the full parameter object structurally, not a JSON string.
- When `${parameters}` is embedded inside a larger string, the parameter object is JSON-serialized into that string.
- The template layer does not add cross-layer type validation beyond existing primitive behavior.
- Composed tools delegate only to primitives in this issue; composed-to-composed chaining is out of scope.
- Pack validation rejects invalid placeholder syntax, undeclared parameter references, invalid parameter schema, and self-referencing primitive names.
- Tool events record both the composed tool identity and the resolved primitive details, including `failure_origin` values `resolution`, `primitive`, or `delegation`.

## Unit Tests
- `backend/internal/engine/template_test.go`
  - parameter substitution into nested objects and strings
  - secret substitution at build time
  - dot-notation traversal success and missing-path errors
  - exact `${parameters}` structural replacement
  - embedded `${parameters}` string serialization
  - single-pass non-recursive behavior
  - placeholder syntax validation and unknown-placeholder handling
- `backend/internal/engine/tool_registry_test.go`
  - composed tool registration from manifest config
  - soft-disable when primitive is missing
  - build-time secret resolution behavior
  - composed execution delegates to the primitive with resolved args
  - failure paths surface clean composed-tool-scoped errors
- `challengepack` validation tests
  - invalid composed-tool placeholder references fail validation
  - self-referencing primitive names fail validation
  - valid composed-tool definitions pass validation

## Integration / Functional Tests
- Registry build plus composed tool execution through a real primitive path succeeds end to end.
- Telemetry emitted from composed-tool execution includes composed-tool metadata, resolved primitive metadata, and `failure_origin`.
- Existing mock-tool behavior continues to pass using the shared template helpers.

## Smoke Tests
- `go test ./backend/internal/engine/...`
- `go test ./backend/internal/worker/...`
- `go test ./challengepack/...`

## E2E Tests
- N/A - this issue adds engine and validation behavior, not a user-facing HTTP flow.

## Manual / cURL Tests
- N/A - no HTTP contract is introduced in this issue.
- Reviewer manual verification commands:
```bash
go test ./backend/internal/engine/... ./backend/internal/worker/... ./challengepack/...
```
