package throttle

import (
	"context"
	"errors"
	"time"
	"unicode/utf8"

	"github.com/agentclash/agentclash/runtime/provider"
)

// Metrics is a Fleet 14 hook for provider throttle counters.
type Metrics interface {
	Request(providerKey string)
	ThrottleWait(providerKey string)
	RateLimit(providerKey string)
	// Cooldown records that a provider entered a Retry-After cooldown window.
	Cooldown(providerKey string, d time.Duration)
}

// NoopMetrics discards throttle metric events.
type NoopMetrics struct{}

func (NoopMetrics) Request(string)                 {}
func (NoopMetrics) ThrottleWait(string)            {}
func (NoopMetrics) RateLimit(string)               {}
func (NoopMetrics) Cooldown(string, time.Duration) {}

// Client decorates a provider.Client (and optional StreamingClient) with
// outbound throttling.
type Client struct {
	inner   provider.Client
	limiter Limiter
	cfg     Config
	metrics Metrics
}

// Wrap returns inner unchanged when no provider limits are enabled.
func Wrap(inner provider.Client, limiter Limiter, cfg Config, metrics ...Metrics) provider.Client {
	if inner == nil || limiter == nil || !cfgHasLimits(cfg) {
		return inner
	}
	var m Metrics = NoopMetrics{}
	if len(metrics) > 0 && metrics[0] != nil {
		m = metrics[0]
	}
	return &Client{inner: inner, limiter: limiter, cfg: cfg, metrics: m}
}

func cfgHasLimits(cfg Config) bool {
	for _, lim := range cfg.LimitsByProvider {
		if lim.Enabled() {
			return true
		}
	}
	return false
}

func (c *Client) InvokeModel(ctx context.Context, request provider.Request) (provider.Response, error) {
	return c.invoke(ctx, request, func(ctx context.Context) (provider.Response, error) {
		return c.inner.InvokeModel(ctx, request)
	})
}

// StreamModel implements provider.StreamingClient when the inner client does.
func (c *Client) StreamModel(ctx context.Context, request provider.Request, onDelta func(provider.StreamDelta) error) (provider.Response, error) {
	streaming, ok := c.inner.(provider.StreamingClient)
	if !ok {
		return provider.Response{}, provider.NewFailure(
			request.ProviderKey,
			provider.FailureCodeUnsupportedCapability,
			"inner client does not support streaming",
			false,
			provider.ErrStreamingNotSupported,
		)
	}
	return c.invoke(ctx, request, func(ctx context.Context) (provider.Response, error) {
		return streaming.StreamModel(ctx, request, onDelta)
	})
}

// ListModels passes through without throttling (enumeration is not model QPS).
func (c *Client) ListModels(ctx context.Context, request provider.ListModelsRequest) ([]provider.ModelInfo, error) {
	lister, ok := c.inner.(provider.ModelLister)
	if !ok {
		return nil, provider.NewFailure(
			request.ProviderKey,
			provider.FailureCodeUnsupportedCapability,
			"inner client does not support model listing",
			false,
			provider.ErrListModelsNotSupported,
		)
	}
	return lister.ListModels(ctx, request)
}

func (c *Client) invoke(ctx context.Context, request provider.Request, call func(context.Context) (provider.Response, error)) (provider.Response, error) {
	key := Key{Provider: request.ProviderKey, Credential: request.CredentialReference}
	limits := c.cfg.limitsFor(request.ProviderKey)
	if !limits.Enabled() {
		return call(ctx)
	}
	if c.metrics == nil {
		c.metrics = NoopMetrics{}
	}
	c.metrics.Request(request.ProviderKey)

	est := c.cfg.estimate(messageChars(request))
	lease, err := c.limiter.Acquire(ctx, key, est)
	if err != nil {
		if errors.Is(err, ErrAcquireTimeout) {
			c.metrics.ThrottleWait(request.ProviderKey)
			return provider.Response{}, provider.NewFailure(
				request.ProviderKey,
				provider.FailureCodeRateLimit,
				"provider throttle acquire timed out",
				true,
				err,
			)
		}
		return provider.Response{}, err
	}
	defer lease.Release()

	resp, err := call(ctx)
	if err != nil {
		if failure, ok := provider.AsFailure(err); ok && failure.Code == provider.FailureCodeRateLimit {
			c.metrics.RateLimit(request.ProviderKey)
			if failure.RetryAfter > 0 {
				c.limiter.CoolDown(key, failure.RetryAfter)
				c.metrics.Cooldown(request.ProviderKey, failure.RetryAfter)
			}
		}
		return provider.Response{}, err
	}
	actual := resp.Usage.TotalTokens
	if actual == 0 {
		actual = resp.Usage.InputTokens + resp.Usage.OutputTokens
	}
	lease.Reconcile(actual)
	return resp, nil
}

func messageChars(request provider.Request) int {
	n := 0
	for _, msg := range request.Messages {
		n += utf8.RuneCountInString(msg.Content)
	}
	return n
}

// WrapRouter returns a Router with each adapter wrapped by Wrap.
func WrapRouter(router provider.Router, limiter Limiter, cfg Config, metrics ...Metrics) provider.Router {
	if limiter == nil || !cfgHasLimits(cfg) {
		return router
	}
	return router.WithClientWrapper(func(_ string, client provider.Client) provider.Client {
		return Wrap(client, limiter, cfg, metrics...)
	})
}
