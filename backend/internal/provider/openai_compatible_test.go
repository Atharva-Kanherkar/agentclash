package provider

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type staticCredentialResolver struct {
	value string
	err   error
}

func (s staticCredentialResolver) Resolve(context.Context, string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.value, nil
}

func TestOpenAICompatibleClientNormalizesSuccess(t *testing.T) {
	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
				t.Fatalf("authorization header = %q, want bearer token", got)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}
			if len(body) == 0 {
				t.Fatalf("expected request body")
			}

			return jsonResponse(http.StatusOK, `{
			"model":"gpt-4.1",
			"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"native step output"}}],
			"usage":{"prompt_tokens":11,"completion_tokens":7,"total_tokens":18}
		}`), nil
		}),
	}

	client := NewOpenAICompatibleClient(httpClient, "https://example.com/v1", staticCredentialResolver{value: "test-key"})

	response, err := client.InvokeModel(context.Background(), Request{
		ProviderKey:         "openai",
		CredentialReference: "env://OPENAI_API_KEY",
		Model:               "gpt-4.1",
		StepTimeout:         time.Second,
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
	})
	if err != nil {
		t.Fatalf("InvokeModel returned error: %v", err)
	}
	if response.OutputText != "native step output" {
		t.Fatalf("output text = %q, want native step output", response.OutputText)
	}
	if response.Usage.TotalTokens != 18 {
		t.Fatalf("total tokens = %d, want 18", response.Usage.TotalTokens)
	}
}

func TestOpenAICompatibleClientNormalizesRateLimitFailure(t *testing.T) {
	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusTooManyRequests, `{"error":{"message":"too many requests"}}`), nil
		}),
	}

	client := NewOpenAICompatibleClient(httpClient, "https://example.com/v1", staticCredentialResolver{value: "test-key"})

	_, err := client.InvokeModel(context.Background(), Request{
		ProviderKey:         "openai",
		CredentialReference: "env://OPENAI_API_KEY",
		Model:               "gpt-4.1",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
	})
	if err == nil {
		t.Fatalf("expected error")
	}

	failure, ok := AsFailure(err)
	if !ok {
		t.Fatalf("expected provider failure, got %T", err)
	}
	if failure.Code != FailureCodeRateLimit {
		t.Fatalf("failure code = %s, want %s", failure.Code, FailureCodeRateLimit)
	}
	if !failure.Retryable {
		t.Fatalf("rate limit failure should be retryable")
	}
}

func TestOpenAICompatibleClientNormalizesMalformedResponse(t *testing.T) {
	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{"model":"gpt-4.1","choices":[]}`), nil
		}),
	}

	client := NewOpenAICompatibleClient(httpClient, "https://example.com/v1", staticCredentialResolver{value: "test-key"})

	_, err := client.InvokeModel(context.Background(), Request{
		ProviderKey:         "openai",
		CredentialReference: "env://OPENAI_API_KEY",
		Model:               "gpt-4.1",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
	})
	if err == nil {
		t.Fatalf("expected error")
	}

	failure, ok := AsFailure(err)
	if !ok {
		t.Fatalf("expected provider failure, got %T", err)
	}
	if failure.Code != FailureCodeMalformedResponse {
		t.Fatalf("failure code = %s, want %s", failure.Code, FailureCodeMalformedResponse)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func jsonResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(strings.NewReader(body)),
	}
}
