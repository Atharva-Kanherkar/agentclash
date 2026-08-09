package engine

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/agentclash/agentclash/runtime/provider"
)

func (e NativeExecutor) invokeWithRetries(ctx context.Context, request provider.Request) (provider.Response, error) {
	backoff := e.initialRetryBackoff
	if backoff <= 0 {
		backoff = defaultRetryBackoff
	}
	attempts := e.maxRetryAttempts
	if attempts <= 0 {
		attempts = defaultRetryAttempts
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		response, err := e.invokeModel(ctx, request)
		if err == nil {
			return response, nil
		}

		failure, ok := provider.AsFailure(err)
		if !ok || !failure.Retryable || !isTransientProviderCode(failure.Code) || attempt == attempts {
			return provider.Response{}, err
		}

		lastErr = err
		wait := e.retryBackoff(failure, backoff)
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return provider.Response{}, ctx.Err()
		case <-timer.C:
		}
		backoff *= 2
	}

	return provider.Response{}, lastErr
}

func (e NativeExecutor) invokeModel(ctx context.Context, request provider.Request) (provider.Response, error) {
	streamingClient, ok := e.client.(provider.StreamingClient)
	if !ok {
		return e.client.InvokeModel(ctx, request)
	}

	return streamingClient.StreamModel(ctx, request, func(delta provider.StreamDelta) error {
		if observerErr := e.observer.OnProviderOutput(ctx, request, delta); observerErr != nil {
			return NewFailure(StopReasonObserverError, "record native provider output event", observerErr)
		}
		return nil
	})
}

func isTransientProviderCode(code provider.FailureCode) bool {
	return code == provider.FailureCodeRateLimit ||
		code == provider.FailureCodeTimeout ||
		code == provider.FailureCodeUnavailable
}

func (e NativeExecutor) retryBackoff(failure provider.Failure, baseBackoff time.Duration) time.Duration {
	var wait time.Duration
	switch {
	case failure.RetryAfter > 0:
		wait = failure.RetryAfter + 1*time.Second
	case failure.Code == provider.FailureCodeRateLimit && baseBackoff < rateLimitMinBackoff:
		wait = rateLimitMinBackoff
	default:
		wait = baseBackoff
	}
	jitter := e.retryJitter
	if jitter == nil {
		jitter = defaultRetryJitter
	}
	return jitter(wait)
}

// defaultRetryJitter spreads synchronized retry waves (±20%).
// Activity-side only — never call from workflow code.
func defaultRetryJitter(wait time.Duration) time.Duration {
	if wait <= 0 {
		return wait
	}
	// Factor in [0.8, 1.2).
	factor := 0.8 + 0.4*rand.Float64()
	return time.Duration(float64(wait) * factor)
}
