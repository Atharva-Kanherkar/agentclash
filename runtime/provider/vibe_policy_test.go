package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestOpenRouterBoundedPolicyOnWire(t *testing.T) {
	seen := ""
	client := NewOpenAICompatibleClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "openai/gpt-4.1-mini" || body["max_tokens"] != float64(2048) || body["temperature"] != float64(0) || body["response_format"].(map[string]any)["type"] != "json_object" {
			t.Fatalf("policy missing from wire: %#v", body)
		}
		policy := body["provider"].(map[string]any)
		if policy["allow_fallbacks"] != false || policy["require_parameters"] != true || policy["only"].([]any)[0] != "openai" || policy["max_price"].(map[string]any)["prompt"] != float64(0.4) {
			t.Fatalf("routing policy changed: %#v", policy)
		}
		return sseResponse(http.StatusOK, "data: {\"id\":\"gen-safe\",\"model\":\"openai/gpt-4.1-mini\",\"choices\":[{\"delta\":{\"content\":\"{}\"},\"finish_reason\":\"stop\"}]}\n\ndata: {\"id\":\"gen-safe\",\"choices\":[],\"usage\":{\"prompt_tokens\":20,\"completion_tokens\":2,\"total_tokens\":22,\"cost\":0.0000112}}\n\ndata: [DONE]\n\n"), nil
	})}, "https://openrouter.ai/api/v1", staticCredentialResolver{value: "local-fake"})
	zero := 0.0
	result, err := client.InvokeModel(context.Background(), Request{ProviderKey: "openrouter", CredentialReference: "test", Model: "openai/gpt-4.1-mini", MaxOutputTokens: 2048, Temperature: &zero, ResponseFormat: json.RawMessage(`{"type":"json_object"}`), OpenRouterPolicy: json.RawMessage(`{"only":["openai"],"allow_fallbacks":false,"require_parameters":true,"max_price":{"prompt":0.4,"completion":1.6}}`), MaxResponseBytes: 4096, Messages: []Message{{Role: "user", Content: "hello"}}, OnGeneration: func(id string) error { seen = id; return nil }})
	if err != nil || seen != "gen-safe" || result.GenerationID != seen || result.Usage.CostUSD == nil || result.Usage.CostUSD.String() != "0.0000112" {
		t.Fatalf("generation/cost lost: %#v %v", result, err)
	}
}

func TestOpenRouterOversizedResponseIsNotSuccess(t *testing.T) {
	client := NewOpenAICompatibleClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: io.NopCloser(strings.NewReader("data: {\"choices\":[{\"delta\":{\"content\":\"" + strings.Repeat("x", 4096) + "\"},\"finish_reason\":\"stop\"}]}\n\n"))}, nil
	})}, "https://openrouter.ai/api/v1", staticCredentialResolver{value: "local-fake"})
	if _, err := client.InvokeModel(context.Background(), Request{ProviderKey: "openrouter", CredentialReference: "test", Model: "openai/gpt-4.1-mini", MaxResponseBytes: 128, Messages: []Message{{Role: "user", Content: "hello"}}}); err == nil {
		t.Fatal("truncated stream became success")
	}
}
