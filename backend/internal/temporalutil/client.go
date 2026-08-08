package temporalutil

import (
	"crypto/tls"
	"os"

	temporalsdk "go.temporal.io/sdk/client"
)

// ClientOption mutates Temporal client options before Dial.
type ClientOption func(*temporalsdk.Options)

// WithMetricsHandler attaches a Temporal MetricsHandler (Fleet 14 OTel bridge).
func WithMetricsHandler(handler temporalsdk.MetricsHandler) ClientOption {
	return func(opts *temporalsdk.Options) {
		if handler != nil {
			opts.MetricsHandler = handler
		}
	}
}

// NewClient creates a Temporal client, automatically enabling TLS and API key
// auth when TEMPORAL_API_KEY is set (required for Temporal Cloud).
func NewClient(hostPort, namespace string, options ...ClientOption) (temporalsdk.Client, error) {
	opts := temporalsdk.Options{
		HostPort:  hostPort,
		Namespace: namespace,
	}

	apiKey := os.Getenv("TEMPORAL_API_KEY")
	if apiKey != "" {
		opts.ConnectionOptions = temporalsdk.ConnectionOptions{
			TLS: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		}
		opts.Credentials = temporalsdk.NewAPIKeyStaticCredentials(apiKey)
	}
	for _, opt := range options {
		if opt != nil {
			opt(&opts)
		}
	}

	return temporalsdk.Dial(opts)
}
