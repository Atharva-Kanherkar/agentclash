package provider

import (
	"context"
	"errors"
	"net/netip"
	"testing"
)

type endpointResolverFunc func(context.Context, string, string) ([]netip.Addr, error)

func (f endpointResolverFunc) LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error) {
	return f(ctx, network, host)
}

func TestNormalizeBaseURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "trim and trailing slash", raw: "  HTTPS://Example.COM:443/v1///  ", want: "https://example.com/v1"},
		{name: "non-default port", raw: "https://Example.COM:8443/openai/v1/", want: "https://example.com:8443/openai/v1"},
		{name: "ipv6 literal", raw: "https://[2606:4700:4700::1111]:443/v1/", want: "https://[2606:4700:4700::1111]/v1"},
		{name: "encoded path semantics", raw: "https://example.com/openai%2Fv1/", want: "https://example.com/openai%2Fv1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeBaseURL(tt.raw)
			if err != nil || got != tt.want {
				t.Fatalf("NormalizeBaseURL(%q) = %q, %v; want %q", tt.raw, got, err, tt.want)
			}
		})
	}
}

func TestNormalizeBaseURLRejectsUnsafeShapes(t *testing.T) {
	tests := []string{
		"",
		"example.com/v1",
		"http://example.com/v1",
		"https://user:pass@example.com/v1",
		"https://example.com/v1?token=secret",
		"https://example.com/v1#fragment",
		"mailto:provider@example.com",
	}
	for _, raw := range tests {
		t.Run(raw, func(t *testing.T) {
			if _, err := NormalizeBaseURL(raw); !IsEndpointValidationError(err) {
				t.Fatalf("NormalizeBaseURL(%q) error = %v, want EndpointValidationError", raw, err)
			}
		})
	}
}

func TestValidateBaseURLRequiresEveryDNSAnswerToBePublic(t *testing.T) {
	resolver := endpointResolverFunc(func(_ context.Context, network, host string) ([]netip.Addr, error) {
		if network != "ip" || host != "models.example.net" {
			t.Fatalf("lookup = %q/%q", network, host)
		}
		return []netip.Addr{
			netip.MustParseAddr("93.184.216.34"),
			netip.MustParseAddr("10.0.0.8"),
		}, nil
	})

	_, err := validateBaseURL(context.Background(), "https://models.example.net/v1", resolver)
	if !IsEndpointValidationError(err) {
		t.Fatalf("validateBaseURL error = %v, want EndpointValidationError", err)
	}
}

func TestValidateBaseURLAcceptsPublicDNSAnswers(t *testing.T) {
	resolver := endpointResolverFunc(func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{
			netip.MustParseAddr("93.184.216.34"),
			netip.MustParseAddr("2606:4700:4700::1111"),
		}, nil
	})
	validated, err := validateBaseURL(context.Background(), "https://models.example.net/v1/", resolver)
	if err != nil {
		t.Fatalf("validateBaseURL: %v", err)
	}
	if validated.baseURL != "https://models.example.net/v1" || len(validated.addresses) != 2 {
		t.Fatalf("validated endpoint = %#v", validated)
	}
}

func TestValidateBaseURLRejectsDNSFailureAndSpecialHostnames(t *testing.T) {
	lookupErr := errors.New("dns unavailable")
	resolver := endpointResolverFunc(func(context.Context, string, string) ([]netip.Addr, error) {
		return nil, lookupErr
	})
	if _, err := validateBaseURL(context.Background(), "https://api.example.net", resolver); !errors.Is(err, lookupErr) {
		t.Fatalf("DNS error = %v, want wrapped lookup error", err)
	}

	called := false
	resolver = endpointResolverFunc(func(context.Context, string, string) ([]netip.Addr, error) {
		called = true
		return nil, nil
	})
	if _, err := validateBaseURL(context.Background(), "https://metadata.google.internal", resolver); !IsEndpointValidationError(err) {
		t.Fatalf("special hostname error = %v", err)
	}
	if called {
		t.Fatal("special-use hostname reached DNS resolver")
	}
}

func TestPublicEndpointAddressPolicy(t *testing.T) {
	public := []string{"8.8.8.8", "93.184.216.34", "2606:4700:4700::1111"}
	for _, raw := range public {
		if !isPublicEndpointAddress(netip.MustParseAddr(raw)) {
			t.Errorf("%s rejected as non-public", raw)
		}
	}

	blocked := []string{
		"0.0.0.0", "10.0.0.1", "100.64.0.1", "127.0.0.1", "169.254.169.254",
		"172.16.0.1", "192.168.1.1", "192.0.2.1", "198.18.0.1", "198.51.100.1",
		"203.0.113.1", "224.0.0.1", "240.0.0.1", "::", "::1", "64:ff9b::1",
		"100::1", "2001:db8::1", "2002::1", "3fff::1", "fc00::1", "fe80::1", "ff02::1",
	}
	for _, raw := range blocked {
		if isPublicEndpointAddress(netip.MustParseAddr(raw)) {
			t.Errorf("%s accepted as public", raw)
		}
	}
}
