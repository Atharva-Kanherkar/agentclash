package provider

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestOpenAICompatibleRequestBaseURLOverridesAdapterDefault(t *testing.T) {
	var gotURL string
	httpClient := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		gotURL = request.URL.String()
		return sseResponse(http.StatusOK, strings.Join([]string{
			`data: {"model":"controlled-model","choices":[{"delta":{"content":"ok"},"finish_reason":"stop"}]}`,
			``,
			`data: [DONE]`,
			``,
		}, "\n")), nil
	})}
	client := NewOpenAICompatibleClient(httpClient, defaultOpenAIBaseURL, staticCredentialResolver{value: "test-key"})
	client.endpointGuard = endpointGuard{
		resolver:                  publicEndpointResolver("93.184.216.34"),
		allowNonStandardTransport: true,
	}

	_, err := client.InvokeModel(context.Background(), Request{
		ProviderKey:         "custom",
		CredentialReference: "env://CUSTOM_API_KEY",
		BaseURL:             "https://models.example.net/openai/v1",
		Model:               "controlled-model",
		Messages:            []Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("InvokeModel: %v", err)
	}
	if gotURL != "https://models.example.net/openai/v1/chat/completions" {
		t.Fatalf("request URL = %q", gotURL)
	}
}

func TestListModelsRequestBaseURLOverridesAdapterDefaults(t *testing.T) {
	tests := []struct {
		name     string
		wantURL  string
		response string
		list     func(*http.Client, endpointGuard) ([]ModelInfo, error)
	}{
		{
			name:     "openai compatible",
			wantURL:  "https://models.example.net/base/models",
			response: `{"data":[]}`,
			list: func(httpClient *http.Client, guard endpointGuard) ([]ModelInfo, error) {
				client := NewOpenAICompatibleClient(httpClient, defaultOpenAIBaseURL, staticCredentialResolver{value: "key"})
				client.endpointGuard = guard
				return client.ListModels(context.Background(), ListModelsRequest{ProviderKey: "custom", CredentialReference: "ref", BaseURL: "https://models.example.net/base"})
			},
		},
		{
			name:     "anthropic",
			wantURL:  "https://models.example.net/base/v1/models?limit=1000",
			response: `{"data":[]}`,
			list: func(httpClient *http.Client, guard endpointGuard) ([]ModelInfo, error) {
				client := NewAnthropicClient(httpClient, defaultAnthropicBaseURL, "", staticCredentialResolver{value: "key"})
				client.endpointGuard = guard
				return client.ListModels(context.Background(), ListModelsRequest{ProviderKey: "anthropic", CredentialReference: "ref", BaseURL: "https://models.example.net/base"})
			},
		},
		{
			name:     "gemini",
			wantURL:  "https://models.example.net/base/v1beta/models?pageSize=1000&key=key",
			response: `{"models":[]}`,
			list: func(httpClient *http.Client, guard endpointGuard) ([]ModelInfo, error) {
				client := NewGeminiClient(httpClient, defaultGeminiBaseURL, staticCredentialResolver{value: "key"})
				client.endpointGuard = guard
				return client.ListModels(context.Background(), ListModelsRequest{ProviderKey: "gemini", CredentialReference: "ref", BaseURL: "https://models.example.net/base"})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotURL string
			httpClient := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				gotURL = request.URL.String()
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.response)),
					Request:    request,
				}, nil
			})}
			_, err := tt.list(httpClient, endpointGuard{
				resolver:                  publicEndpointResolver("93.184.216.34"),
				allowNonStandardTransport: true,
			})
			if err != nil {
				t.Fatalf("ListModels: %v", err)
			}
			if gotURL != tt.wantURL {
				t.Fatalf("request URL = %q, want %q", gotURL, tt.wantURL)
			}
		})
	}
}

func TestDefaultRouterRegistersCustomProvider(t *testing.T) {
	router := NewDefaultRouter(&http.Client{}, staticCredentialResolver{value: "key"})
	if _, ok := router.adapters["custom"].(OpenAICompatibleClient); !ok {
		t.Fatalf("custom adapter = %T, want OpenAICompatibleClient", router.adapters["custom"])
	}
}
