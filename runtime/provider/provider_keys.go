package provider

import (
	"fmt"
	"strings"
)

var supportedProviderKeys = []string{
	"openai",
	"anthropic",
	"gemini",
	"xai",
	"openrouter",
	"mistral",
	"custom",
}

// NormalizeProviderKey canonicalizes and validates a provider identifier used
// by provider accounts. Keeping this list next to the provider contracts avoids
// API and runtime allowlists drifting apart.
func NormalizeProviderKey(raw string) (string, error) {
	key := strings.ToLower(strings.TrimSpace(raw))
	for _, supported := range supportedProviderKeys {
		if key == supported {
			return key, nil
		}
	}
	return "", fmt.Errorf("%w: provider_key must be one of %s", ErrUnsupportedProvider, strings.Join(supportedProviderKeys, ", "))
}

// SupportedProviderKeys returns the canonical account-provider keys.
func SupportedProviderKeys() []string {
	return append([]string(nil), supportedProviderKeys...)
}
