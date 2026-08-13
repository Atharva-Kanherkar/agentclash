package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/agentclash/agentclash/backend/internal/budget"
	"github.com/agentclash/agentclash/backend/internal/repository"
	"github.com/agentclash/agentclash/runtime/provider"
	"github.com/google/uuid"
)

// Shared plumbing for the api-server's synchronous, single-shot LLM features
// (run_ranking_insights.go, challenge_pack_generate.go). Each feature owns its
// own prompt, output contract, and typed errors. What they share is mechanical:
// how a model's JSON reply is decoded, how the workspace's spend policies gate
// the call, and how a provider failure maps onto HTTP.

// decodeStrictJSONObject decodes a model reply into target. Models are asked
// for bare JSON but sometimes wrap it in a markdown fence or bracket it with
// prose, so the raw text, the de-fenced text, and the first balanced JSON
// object are each tried in turn. Decoding is strict — unknown fields and
// trailing content are rejected — so a reply that quietly renames or invents a
// field fails loudly instead of being silently dropped.
//
// A candidate must actually be a JSON object. Without that check the literal
// `null` decodes into any struct as its zero value with no error, so a model
// that answered "null" would be reported as a successful reply carrying an
// empty result. `{}` is left alone: it is an object, and every caller's own
// content validation rejects it on the merits.
func decodeStrictJSONObject[T any](raw string, target *T) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return errors.New("response did not contain a JSON object")
	}

	candidates := []string{trimmed}
	if withoutFence, ok := stripJSONFence(trimmed); ok {
		candidates = append(candidates, withoutFence)
	}
	if extracted, ok := extractFirstJSONObject(trimmed); ok {
		candidates = append(candidates, extracted)
	}

	var decodeErr error
	for _, candidate := range candidates {
		if !strings.HasPrefix(strings.TrimSpace(candidate), "{") {
			decodeErr = errors.New("response did not contain a JSON object")
			continue
		}
		decoder := json.NewDecoder(strings.NewReader(candidate))
		decoder.DisallowUnknownFields()
		var decoded T
		if err := decoder.Decode(&decoded); err != nil {
			decodeErr = err
			continue
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			decodeErr = errors.New("response contained trailing content after JSON object")
			continue
		}
		*target = decoded
		return nil
	}

	if decodeErr != nil {
		return decodeErr
	}
	return errors.New("response did not contain a JSON object")
}

func stripJSONFence(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "```") {
		return "", false
	}

	lines := strings.Split(trimmed, "\n")
	if len(lines) < 3 {
		return "", false
	}
	if !strings.HasPrefix(lines[0], "```") || strings.TrimSpace(lines[len(lines)-1]) != "```" {
		return "", false
	}
	return strings.TrimSpace(strings.Join(lines[1:len(lines)-1], "\n")), true
}

func extractFirstJSONObject(raw string) (string, bool) {
	inString := false
	escaped := false
	depth := 0
	start := -1

	for idx, r := range raw {
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == '"' {
				inString = false
			}
			continue
		}

		switch r {
		case '"':
			inString = true
		case '{':
			if depth == 0 {
				start = idx
			}
			depth++
		case '}':
			if depth == 0 {
				continue
			}
			depth--
			if depth == 0 && start >= 0 {
				return raw[start : idx+1], true
			}
		}
	}

	return "", false
}

// sanitizePromptText neutralises the characters an untrusted string would need
// to forge a closing </user_data> tag or a markdown fence inside the prompt.
func sanitizePromptText(value string) string {
	replacer := strings.NewReplacer("<", "(", ">", ")", "`", "'")
	return replacer.Replace(strings.TrimSpace(value))
}

// spendPolicyLister is the slice of a repository the spend guard needs.
type spendPolicyLister interface {
	ListSpendPoliciesByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) ([]repository.SpendPolicyRow, error)
}

// workspaceSpendAllowed reports whether an LLM feature may spend on behalf of a
// workspace: every spend policy attached to the workspace must still be within
// budget. label names the feature in logs. An error means the check itself
// failed and the caller must not proceed either.
func workspaceSpendAllowed(ctx context.Context, lister spendPolicyLister, checker budget.BudgetChecker, workspaceID uuid.UUID, label string) (bool, error) {
	spendPolicies, err := lister.ListSpendPoliciesByWorkspaceID(ctx, workspaceID)
	if err != nil {
		return false, fmt.Errorf("list workspace spend policies: %w", err)
	}

	for _, spendPolicy := range spendPolicies {
		result, err := checker.CheckPreRunBudget(ctx, workspaceID, spendPolicy.ID)
		if err != nil {
			return false, fmt.Errorf("check spend policy budget: %w", err)
		}
		if !result.Allowed {
			return false, nil
		}
		if result.SoftLimitHit {
			slog.Default().Warn("llm feature spend policy soft limit reached",
				"feature", label,
				"workspace_id", workspaceID,
				"spend_policy_id", spendPolicy.ID,
				"current_spend", result.CurrentSpend,
			)
		}
	}

	return true, nil
}

// providerAccountVisibleToWorkspace reports whether a BYOK provider account may
// be spent on behalf of a workspace: it must belong to that workspace and still
// be active. Every LLM feature checks this before preparing credentials.
func providerAccountVisibleToWorkspace(account repository.ProviderAccountRow, workspaceID uuid.UUID) bool {
	return account.WorkspaceID != nil && *account.WorkspaceID == workspaceID && account.Status == "active"
}

// writeRetryAfterError writes an error response that tells the caller when to
// come back, for the two throttled paths an LLM feature has: its own
// per-workspace bucket, and the upstream provider's.
func writeRetryAfterError(w http.ResponseWriter, status int, code string, message string, retryAfter time.Duration) {
	if retryAfter > 0 {
		retryAfterSeconds := int(retryAfter.Seconds())
		if retryAfterSeconds < 1 {
			retryAfterSeconds = 1
		}
		w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfterSeconds))
	}
	writeError(w, status, code, message)
}

// writeProviderFailure maps a provider-side failure onto the HTTP response for
// an LLM feature. feature namespaces the machine-readable error codes
// (e.g. "ranking_insights"); label is the human phrase used in messages.
func writeProviderFailure(w http.ResponseWriter, feature, label string, failure provider.Failure) {
	switch failure.Code {
	case provider.FailureCodeAuth, provider.FailureCodeCredentialUnavailable:
		writeError(w, http.StatusBadRequest, "invalid_provider_credentials", "selected provider credentials are unavailable or invalid")
	case provider.FailureCodeRateLimit:
		writeRetryAfterError(w, http.StatusTooManyRequests, feature+"_provider_rate_limited", "selected provider is rate limited; retry later", failure.RetryAfter)
	case provider.FailureCodeUnsupportedProvider, provider.FailureCodeUnsupportedCapability, provider.FailureCodeInvalidRequest:
		writeError(w, http.StatusBadRequest, "invalid_"+feature+"_provider_request", "selected provider configuration is not supported for "+label)
	case provider.FailureCodeTimeout, provider.FailureCodeUnavailable:
		writeRetryAfterError(w, http.StatusServiceUnavailable, feature+"_provider_unavailable", "selected provider is temporarily unavailable", failure.RetryAfter)
	default:
		writeError(w, http.StatusBadGateway, feature+"_provider_error", label+" provider returned an invalid response")
	}
}
