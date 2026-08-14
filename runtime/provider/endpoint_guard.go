package provider

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

type endpointDialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

type endpointGuard struct {
	resolver                  EndpointResolver
	dialer                    endpointDialer
	allowNonStandardTransport bool
}

func (g endpointGuard) withDefaults() endpointGuard {
	if g.resolver == nil {
		g.resolver = net.DefaultResolver
	}
	if g.dialer == nil {
		g.dialer = &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	}
	return g
}

func effectiveProviderEndpoint(
	ctx context.Context,
	providerKey string,
	override string,
	defaultBaseURL string,
	httpClient *http.Client,
	guard endpointGuard,
) (string, *http.Client, error) {
	if strings.TrimSpace(override) == "" {
		if providerKey == "custom" {
			return "", nil, NewFailure(
				providerKey,
				FailureCodeInvalidRequest,
				"custom provider requires base_url",
				false,
				nil,
			)
		}
		return strings.TrimRight(defaultBaseURL, "/"), httpClient, nil
	}

	guard = guard.withDefaults()
	validated, err := validateBaseURL(ctx, override, guard.resolver)
	if err != nil {
		return "", nil, endpointValidationFailure(providerKey, err)
	}
	guardedClient, err := guard.clientFor(validated, httpClient)
	if err != nil {
		return "", nil, endpointValidationFailure(providerKey, err)
	}
	return validated.baseURL, guardedClient, nil
}

func endpointValidationFailure(providerKey string, err error) error {
	if failure, ok := AsFailure(err); ok {
		return failure
	}
	return NewFailure(
		providerKey,
		FailureCodeInvalidRequest,
		"invalid provider base_url: "+err.Error(),
		false,
		err,
	)
}

func classifyEndpointTransportError(providerKey string, err error) error {
	if IsEndpointValidationError(err) {
		return endpointValidationFailure(providerKey, err)
	}
	return classifyTransportError(providerKey, err)
}

func (g endpointGuard) clientFor(validated validatedEndpoint, base *http.Client) (*http.Client, error) {
	if base == nil {
		base = NewDefaultHTTPClient()
	}
	client := *base
	origin := endpointOriginFromURL(validated.parsedURL)

	previousRedirect := base.CheckRedirect
	client.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if request.URL.User != nil || endpointOriginFromURL(request.URL) != origin {
			return endpointValidationError("provider endpoint redirect must remain on the validated origin", nil)
		}
		if len(via) >= 10 {
			return endpointValidationError("provider endpoint stopped after 10 redirects", nil)
		}
		if previousRedirect != nil {
			return previousRedirect(request, via)
		}
		return nil
	}

	switch transport := base.Transport.(type) {
	case nil:
		cloned := http.DefaultTransport.(*http.Transport).Clone()
		configureGuardedTransport(cloned, validated, g)
		client.Transport = cloned
	case *http.Transport:
		cloned := transport.Clone()
		configureGuardedTransport(cloned, validated, g)
		client.Transport = cloned
	default:
		if !g.allowNonStandardTransport {
			return nil, endpointValidationError("provider endpoint overrides require a standard HTTP transport", nil)
		}
		client.Transport = guardedRoundTripper{next: transport, origin: origin}
	}
	return &client, nil
}

func configureGuardedTransport(transport *http.Transport, validated validatedEndpoint, guard endpointGuard) {
	// Proxies and custom TLS dialers can resolve or connect independently of
	// DialContext. Disable them for untrusted endpoint overrides so every
	// connection is pinned to the address set validated immediately above.
	transport.Proxy = nil
	transport.DialTLS = nil
	transport.DialTLSContext = nil
	transport.DialContext = guardedDialContext(validated, guard.dialer)
	// A fresh guarded transport is built for each invocation so DNS is checked
	// again on the next call. Do not retain an idle connection that could skip
	// that use-time validation or leak transports over a long-running worker.
	transport.DisableKeepAlives = true

	tlsConfig := transport.TLSClientConfig
	if tlsConfig == nil {
		tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	} else {
		tlsConfig = tlsConfig.Clone()
		if tlsConfig.MinVersion == 0 {
			tlsConfig.MinVersion = tls.VersionTLS12
		}
	}
	tlsConfig.ServerName = validated.parsedURL.Hostname()
	tlsConfig.InsecureSkipVerify = false
	transport.TLSClientConfig = tlsConfig
}

func guardedDialContext(validated validatedEndpoint, dialer endpointDialer) func(context.Context, string, string) (net.Conn, error) {
	origin := endpointOriginFromURL(validated.parsedURL)
	addresses := append([]netip.Addr(nil), validated.addresses...)
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, endpointValidationError("provider endpoint dial address is invalid", err)
		}
		if canonicalHost(host) != origin.host || port != origin.port {
			return nil, endpointValidationError("provider endpoint dial escaped the validated origin", nil)
		}

		var dialErrors []error
		for _, candidate := range addresses {
			candidate = candidate.Unmap()
			if network == "tcp4" && !candidate.Is4() {
				continue
			}
			if network == "tcp6" && !candidate.Is6() {
				continue
			}
			connection, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(candidate.String(), port))
			if dialErr == nil {
				return connection, nil
			}
			dialErrors = append(dialErrors, dialErr)
		}
		if len(dialErrors) == 0 {
			return nil, endpointValidationError("provider endpoint has no validated address for the requested network", nil)
		}
		return nil, fmt.Errorf("dial validated provider endpoint: %w", errors.Join(dialErrors...))
	}
}

type endpointOrigin struct {
	scheme string
	host   string
	port   string
}

func endpointOriginFromURL(value *url.URL) endpointOrigin {
	port := value.Port()
	if port == "" {
		switch strings.ToLower(value.Scheme) {
		case "https":
			port = "443"
		case "http":
			port = "80"
		}
	}
	return endpointOrigin{
		scheme: strings.ToLower(value.Scheme),
		host:   canonicalHost(value.Hostname()),
		port:   port,
	}
}

func canonicalHost(host string) string {
	host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	if address, err := netip.ParseAddr(host); err == nil {
		return address.Unmap().String()
	}
	return host
}

type guardedRoundTripper struct {
	next   http.RoundTripper
	origin endpointOrigin
}

func (t guardedRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.URL.User != nil || endpointOriginFromURL(request.URL) != t.origin {
		return nil, endpointValidationError("provider request escaped the validated origin", nil)
	}
	return t.next.RoundTrip(request)
}
