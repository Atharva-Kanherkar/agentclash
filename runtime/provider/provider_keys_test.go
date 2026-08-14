package provider

import (
	"errors"
	"reflect"
	"testing"
)

func TestNormalizeProviderKey(t *testing.T) {
	for _, key := range []string{"openai", "anthropic", "gemini", "xai", "openrouter", "mistral", "custom"} {
		t.Run(key, func(t *testing.T) {
			got, err := NormalizeProviderKey("  " + key + "  ")
			if err != nil || got != key {
				t.Fatalf("NormalizeProviderKey() = %q, %v; want %q, nil", got, err, key)
			}
		})
	}

	got, err := NormalizeProviderKey(" CuStOm ")
	if err != nil || got != "custom" {
		t.Fatalf("case normalization = %q, %v", got, err)
	}
	if _, err := NormalizeProviderKey("azure-openai"); !errors.Is(err, ErrUnsupportedProvider) {
		t.Fatalf("unknown provider error = %v, want ErrUnsupportedProvider", err)
	}
}

func TestSupportedProviderKeysReturnsCopy(t *testing.T) {
	want := []string{"openai", "anthropic", "gemini", "xai", "openrouter", "mistral", "custom"}
	got := SupportedProviderKeys()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SupportedProviderKeys() = %#v, want %#v", got, want)
	}
	got[0] = "mutated"
	if SupportedProviderKeys()[0] != "openai" {
		t.Fatal("SupportedProviderKeys returned shared storage")
	}
}
