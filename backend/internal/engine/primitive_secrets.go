package engine

import "strings"

// secretSafePrimitives enumerates every primitive that is hardened to
// handle plaintext ${secrets.*} values inside the sandbox without
// exposing them to the evaluated agent. Adding a primitive to this set
// is a security-relevant change and should be reviewed carefully:
//
//   - the primitive must NOT write its resolved args to a sandbox
//     filesystem path the agent can read (use stdin pipes or private
//     filesystem roots not in ReadableRoots).
//   - the primitive must NOT place the resolved secret in argv — argv
//     is observable via /proc/[pid]/cmdline by any process in the
//     sandbox.
//   - the primitive must strip sensitive response/output fields
//     (Authorization / Cookie / X-API-Key headers, etc.) before
//     returning a ToolExecutionResult to the LLM.
//   - the primitive must never include a resolved secret in an error
//     message.
//
// See issue #186 for the full threat model.
var secretSafePrimitives = map[string]struct{}{
	httpRequestToolName: {},
}

func primitiveAcceptsSecrets(primitiveName string) bool {
	_, ok := secretSafePrimitives[primitiveName]
	return ok
}

// templateReferencesSecrets walks any template value (map / slice /
// string) and reports whether at least one string element references
// a ${secrets.*} placeholder. Used at composed-tool build time to
// gate secret-bearing args to primitives that can handle them safely.
func templateReferencesSecrets(value any) bool {
	switch v := value.(type) {
	case string:
		return stringReferencesSecrets(v)
	case map[string]any:
		for _, inner := range v {
			if templateReferencesSecrets(inner) {
				return true
			}
		}
	case []any:
		for _, inner := range v {
			if templateReferencesSecrets(inner) {
				return true
			}
		}
	}
	return false
}

func stringReferencesSecrets(s string) bool {
	remaining := s
	for {
		idx := strings.Index(remaining, "${")
		if idx == -1 {
			return false
		}
		after := remaining[idx+2:]
		closeIdx := strings.Index(after, "}")
		if closeIdx == -1 {
			return false
		}
		if strings.HasPrefix(after[:closeIdx], "secrets.") {
			return true
		}
		remaining = after[closeIdx+1:]
	}
}

// sensitiveResponseHeaders is the case-insensitive denylist of HTTP
// response header names that may carry authentication material echoed
// back by a remote API. When http_request returns its parsed response
// to the LLM, these headers are replaced with a redacted marker so a
// server that mirrors the request Authorization header (for debug or
// by accident) cannot leak a ${secrets.X}-substituted value back into
// the agent's context.
//
// The list is intentionally curated rather than heuristic ("strip any
// header containing 'auth'"): a fixed allowlist gives the security
// reviewer a single place to audit, and a heuristic would
// false-positive on legitimate headers like X-Auth-Request-Redirect.
var sensitiveResponseHeaders = map[string]struct{}{
	"authorization":       {},
	"proxy-authorization": {},
	"www-authenticate":    {},
	"proxy-authenticate":  {},
	"cookie":              {},
	"set-cookie":          {},
	"x-api-key":           {},
	"x-auth-token":        {},
	"x-access-token":      {},
	"x-amz-security-token": {},
}

const redactedHeaderMarker = "[redacted]"

// scrubSensitiveResponseHeaders walks a decoded http_request response
// payload and replaces any sensitive header value with a redacted
// marker. Safe to call on any shape — a nil, a non-map, or a response
// without headers is a no-op.
func scrubSensitiveResponseHeaders(payload any) {
	m, ok := payload.(map[string]any)
	if !ok {
		return
	}
	headers, ok := m["headers"].(map[string]any)
	if !ok {
		return
	}
	for key := range headers {
		if _, sensitive := sensitiveResponseHeaders[strings.ToLower(strings.TrimSpace(key))]; sensitive {
			headers[key] = redactedHeaderMarker
		}
	}
}
