package provider

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"testing"
)

type endpointDialerFunc func(context.Context, string, string) (net.Conn, error)

func (f endpointDialerFunc) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return f(ctx, network, address)
}

func publicEndpointResolver(addresses ...string) EndpointResolver {
	parsed := make([]netip.Addr, len(addresses))
	for i, address := range addresses {
		parsed[i] = netip.MustParseAddr(address)
	}
	return endpointResolverFunc(func(context.Context, string, string) ([]netip.Addr, error) {
		return append([]netip.Addr(nil), parsed...), nil
	})
}

func TestGuardedDialUsesOnlyValidatedAddresses(t *testing.T) {
	validated, err := validateBaseURL(
		context.Background(),
		"https://models.example.net/v1",
		publicEndpointResolver("93.184.216.34", "2606:4700:4700::1111"),
	)
	if err != nil {
		t.Fatalf("validateBaseURL: %v", err)
	}

	var dialed []string
	dialer := endpointDialerFunc(func(_ context.Context, _ string, address string) (net.Conn, error) {
		dialed = append(dialed, address)
		client, server := net.Pipe()
		_ = server.Close()
		return client, nil
	})
	client, err := (endpointGuard{dialer: dialer}).clientFor(validated, &http.Client{})
	if err != nil {
		t.Fatalf("clientFor: %v", err)
	}
	transport := client.Transport.(*http.Transport)
	connection, err := transport.DialContext(context.Background(), "tcp", "models.example.net:443")
	if err != nil {
		t.Fatalf("DialContext: %v", err)
	}
	_ = connection.Close()
	if len(dialed) != 1 || dialed[0] != "93.184.216.34:443" {
		t.Fatalf("dialed = %#v, want first validated address", dialed)
	}

	if _, err := transport.DialContext(context.Background(), "tcp", "metadata.google.internal:443"); !IsEndpointValidationError(err) {
		t.Fatalf("escaped dial error = %v, want EndpointValidationError", err)
	}
}

func TestEndpointIsRevalidatedAtUseTime(t *testing.T) {
	addresses := []netip.Addr{netip.MustParseAddr("93.184.216.34")}
	resolver := endpointResolverFunc(func(context.Context, string, string) ([]netip.Addr, error) {
		return append([]netip.Addr(nil), addresses...), nil
	})
	if _, err := ValidateBaseURLWithResolver(context.Background(), "https://models.example.net/v1", resolver); err != nil {
		t.Fatalf("storage-time validation: %v", err)
	}

	addresses = []netip.Addr{netip.MustParseAddr("10.0.0.8")}
	_, _, err := effectiveProviderEndpoint(
		context.Background(),
		"custom",
		"https://models.example.net/v1",
		defaultOpenAIBaseURL,
		&http.Client{},
		endpointGuard{resolver: resolver},
	)
	failure, ok := AsFailure(err)
	if !ok || failure.Code != FailureCodeInvalidRequest || failure.Retryable {
		t.Fatalf("use-time failure = %#v, %v", failure, err)
	}
}

func TestEndpointRedirectPolicy(t *testing.T) {
	tests := []struct {
		name      string
		location  string
		wantError bool
		wantCalls int
	}{
		{name: "same origin", location: "https://models.example.net/v1/next?cursor=1", wantCalls: 2},
		{name: "cross origin", location: "https://attacker.example/v1", wantError: true, wantCalls: 1},
		{name: "scheme downgrade", location: "http://models.example.net/v1", wantError: true, wantCalls: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			base := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return &http.Response{
						StatusCode: http.StatusTemporaryRedirect,
						Header:     http.Header{"Location": []string{tt.location}},
						Body:       io.NopCloser(strings.NewReader("")),
						Request:    request,
					}, nil
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader("ok")),
					Request:    request,
				}, nil
			})}
			_, client, err := effectiveProviderEndpoint(
				context.Background(),
				"custom",
				"https://models.example.net/v1",
				defaultOpenAIBaseURL,
				base,
				endpointGuard{
					resolver:                  publicEndpointResolver("93.184.216.34"),
					allowNonStandardTransport: true,
				},
			)
			if err != nil {
				t.Fatalf("effectiveProviderEndpoint: %v", err)
			}
			response, requestErr := client.Get("https://models.example.net/v1")
			if response != nil {
				_ = response.Body.Close()
			}
			if tt.wantError != (requestErr != nil) {
				t.Fatalf("GET error = %v, wantError %v", requestErr, tt.wantError)
			}
			if tt.wantError && !IsEndpointValidationError(requestErr) {
				t.Fatalf("GET error = %v, want EndpointValidationError", requestErr)
			}
			if calls != tt.wantCalls {
				t.Fatalf("round trips = %d, want %d", calls, tt.wantCalls)
			}
		})
	}
}

func TestCustomProviderWithoutOverrideFailsBeforeCredentialOrHTTP(t *testing.T) {
	credentialCalls := 0
	resolver := credentialResolverFunc(func(context.Context, string) (string, error) {
		credentialCalls++
		return "secret", nil
	})
	httpCalls := 0
	client := NewOpenAICompatibleClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		httpCalls++
		return nil, errors.New("must not be called")
	})}, "", resolver)

	_, err := client.InvokeModel(context.Background(), Request{ProviderKey: "custom", Model: "model"})
	failure, ok := AsFailure(err)
	if !ok || failure.Code != FailureCodeInvalidRequest || failure.Retryable {
		t.Fatalf("InvokeModel error = %v", err)
	}
	if credentialCalls != 0 || httpCalls != 0 {
		t.Fatalf("credential/http calls = %d/%d, want 0/0", credentialCalls, httpCalls)
	}

	_, err = client.ListModels(context.Background(), ListModelsRequest{ProviderKey: "custom"})
	failure, ok = AsFailure(err)
	if !ok || failure.Code != FailureCodeInvalidRequest || failure.Retryable {
		t.Fatalf("ListModels error = %v", err)
	}
	if credentialCalls != 0 || httpCalls != 0 {
		t.Fatalf("list credential/http calls = %d/%d, want 0/0", credentialCalls, httpCalls)
	}
}

type credentialResolverFunc func(context.Context, string) (string, error)

func (f credentialResolverFunc) Resolve(ctx context.Context, reference string) (string, error) {
	return f(ctx, reference)
}
