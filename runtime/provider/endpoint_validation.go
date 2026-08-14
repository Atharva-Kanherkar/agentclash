package provider

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
)

// EndpointResolver is the DNS capability used when validating a provider
// endpoint. net.DefaultResolver satisfies it; the interface also keeps the
// security policy deterministic under test.
type EndpointResolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

// EndpointValidationError is safe to return as a request validation message.
// It deliberately omits resolved addresses so private infrastructure details
// are not reflected to callers.
type EndpointValidationError struct {
	message string
	cause   error
}

func (e *EndpointValidationError) Error() string { return e.message }
func (e *EndpointValidationError) Unwrap() error { return e.cause }

func endpointValidationError(message string, cause error) error {
	return &EndpointValidationError{message: message, cause: cause}
}

// IsEndpointValidationError reports whether err came from endpoint policy
// validation and can therefore be surfaced as a client-side validation error.
func IsEndpointValidationError(err error) bool {
	var validationErr *EndpointValidationError
	return errors.As(err, &validationErr)
}

type validatedEndpoint struct {
	baseURL   string
	parsedURL *url.URL
	addresses []netip.Addr
}

// NormalizeBaseURL performs the context-free portion of endpoint validation.
// Network-backed validation is performed by ValidateBaseURL before storage and
// again by provider clients immediately before dialing.
func NormalizeBaseURL(raw string) (string, error) {
	parsed, err := parseBaseURL(raw)
	if err != nil {
		return "", err
	}
	return parsed.String(), nil
}

// ValidateBaseURL normalizes a provider endpoint and requires its full DNS
// answer set to contain only public, globally routable addresses.
func ValidateBaseURL(ctx context.Context, raw string) (string, error) {
	return ValidateBaseURLWithResolver(ctx, raw, net.DefaultResolver)
}

// ValidateBaseURLWithResolver applies the same storage-time endpoint policy
// with an explicit resolver. Services can inject a deterministic resolver in
// tests while production uses ValidateBaseURL.
func ValidateBaseURLWithResolver(ctx context.Context, raw string, resolver EndpointResolver) (string, error) {
	validated, err := validateBaseURL(ctx, raw, resolver)
	if err != nil {
		return "", err
	}
	return validated.baseURL, nil
}

func validateBaseURL(ctx context.Context, raw string, resolver EndpointResolver) (validatedEndpoint, error) {
	parsed, err := parseBaseURL(raw)
	if err != nil {
		return validatedEndpoint{}, err
	}
	if resolver == nil {
		resolver = net.DefaultResolver
	}

	host := parsed.Hostname()
	addresses := make([]netip.Addr, 0, 2)
	if literal, err := netip.ParseAddr(host); err == nil {
		addresses = append(addresses, literal)
	} else {
		if isSpecialUseHostname(host) {
			return validatedEndpoint{}, endpointValidationError("base_url host must be a public DNS name", nil)
		}
		resolved, lookupErr := resolver.LookupNetIP(ctx, "ip", host)
		if lookupErr != nil {
			return validatedEndpoint{}, endpointValidationError("base_url host could not be resolved", lookupErr)
		}
		addresses = append(addresses, resolved...)
	}

	if len(addresses) == 0 {
		return validatedEndpoint{}, endpointValidationError("base_url host did not resolve to an address", nil)
	}
	for _, address := range addresses {
		if !isPublicEndpointAddress(address) {
			return validatedEndpoint{}, endpointValidationError("base_url host must resolve only to public addresses", nil)
		}
	}

	return validatedEndpoint{
		baseURL:   parsed.String(),
		parsedURL: parsed,
		addresses: append([]netip.Addr(nil), addresses...),
	}, nil
}

func parseBaseURL(raw string) (*url.URL, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, endpointValidationError("base_url is required", nil)
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return nil, endpointValidationError("base_url must be a valid absolute HTTPS URL", err)
	}
	if !parsed.IsAbs() || !strings.EqualFold(parsed.Scheme, "https") || parsed.Host == "" || parsed.Opaque != "" {
		return nil, endpointValidationError("base_url must be an absolute HTTPS URL", nil)
	}
	if parsed.User != nil {
		return nil, endpointValidationError("base_url must not include user information", nil)
	}
	if parsed.RawQuery != "" || parsed.ForceQuery {
		return nil, endpointValidationError("base_url must not include a query", nil)
	}
	if parsed.Fragment != "" || parsed.RawFragment != "" {
		return nil, endpointValidationError("base_url must not include a fragment", nil)
	}

	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if host == "" {
		return nil, endpointValidationError("base_url must include a host", nil)
	}
	port := parsed.Port()
	if port == "443" {
		port = ""
	}
	if literal, literalErr := netip.ParseAddr(host); literalErr == nil {
		if literal.Zone() != "" {
			return nil, endpointValidationError("base_url must not include an address zone", nil)
		}
		host = literal.String()
	}
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	if port != "" {
		host = net.JoinHostPort(strings.Trim(host, "[]"), port)
	}

	parsed.Scheme = "https"
	parsed.Host = host
	if parsed.RawPath != "" {
		rawPath := strings.TrimRight(parsed.EscapedPath(), "/")
		decodedPath, err := url.PathUnescape(rawPath)
		if err != nil {
			return nil, endpointValidationError("base_url path must be valid URL encoding", err)
		}
		parsed.Path = decodedPath
		parsed.RawPath = rawPath
	} else {
		parsed.Path = strings.TrimRight(parsed.Path, "/")
	}
	return parsed, nil
}

func isSpecialUseHostname(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if host == "localhost" || host == "metadata" || host == "instance-data" || host == "metadata.google.internal" {
		return true
	}
	for _, suffix := range []string{".localhost", ".local", ".internal", ".home.arpa", ".test", ".invalid", ".example"} {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}
	return false
}

func isPublicEndpointAddress(address netip.Addr) bool {
	if !address.IsValid() || address.Zone() != "" {
		return false
	}
	address = address.Unmap()
	if !address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() ||
		address.IsLinkLocalUnicast() || address.IsLinkLocalMulticast() ||
		address.IsMulticast() || address.IsUnspecified() {
		return false
	}
	if address.Is6() && !globalIPv6Prefix.Contains(address) {
		return false
	}
	for _, blocked := range blockedEndpointPrefixes {
		if blocked.Contains(address) {
			return false
		}
	}
	return true
}

func mustPrefix(value string) netip.Prefix {
	prefix, err := netip.ParsePrefix(value)
	if err != nil {
		panic(fmt.Sprintf("invalid endpoint prefix %q: %v", value, err))
	}
	return prefix
}

var globalIPv6Prefix = mustPrefix("2000::/3")

// IANA special-purpose address registries, intentionally blocked even for
// entries marked globally reachable. Endpoint overrides are data-plane egress,
// so a conservative public-Internet-only policy is appropriate.
var blockedEndpointPrefixes = []netip.Prefix{
	mustPrefix("0.0.0.0/8"),
	mustPrefix("10.0.0.0/8"),
	mustPrefix("100.64.0.0/10"),
	mustPrefix("127.0.0.0/8"),
	mustPrefix("169.254.0.0/16"),
	mustPrefix("172.16.0.0/12"),
	mustPrefix("192.0.0.0/24"),
	mustPrefix("192.0.2.0/24"),
	mustPrefix("192.31.196.0/24"),
	mustPrefix("192.52.193.0/24"),
	mustPrefix("192.88.99.0/24"),
	mustPrefix("192.168.0.0/16"),
	mustPrefix("192.175.48.0/24"),
	mustPrefix("198.18.0.0/15"),
	mustPrefix("198.51.100.0/24"),
	mustPrefix("203.0.113.0/24"),
	mustPrefix("224.0.0.0/4"),
	mustPrefix("240.0.0.0/4"),
	mustPrefix("::/96"),
	mustPrefix("64:ff9b::/96"),
	mustPrefix("64:ff9b:1::/48"),
	mustPrefix("100::/64"),
	mustPrefix("2001::/23"),
	mustPrefix("2001:db8::/32"),
	mustPrefix("2002::/16"),
	mustPrefix("2620:4f:8000::/48"),
	mustPrefix("3fff::/20"),
	mustPrefix("5f00::/16"),
	mustPrefix("fc00::/7"),
	mustPrefix("fe80::/10"),
	mustPrefix("ff00::/8"),
}
