package simulator_test

import (
	"context"
	"strings"
	"testing"

	"github.com/agentclash/agentclash/backend/internal/repository"
	"github.com/agentclash/agentclash/backend/internal/simulator"
	"github.com/agentclash/agentclash/runtime/provider"
	"github.com/google/uuid"
)

func TestGeneratorForwardsProviderEndpoint(t *testing.T) {
	client := &provider.FakeClient{Response: provider.Response{OutputText: "next question"}}
	generator := simulator.NewGenerator(client)
	message, _, err := generator.GenerateUserMessage(context.Background(), simulator.Input{
		Persona:           "A careful user",
		ProviderKey:       "custom",
		ProviderAccountID: "account-id",
		CredentialRef:     "env://CUSTOM_API_KEY",
		BaseURL:           "https://simulator.example.com/v1",
		Model:             "controlled-model",
	})
	if err != nil {
		t.Fatalf("GenerateUserMessage: %v", err)
	}
	if message != "next question" || len(client.Requests) != 1 {
		t.Fatalf("message/requests = %q/%d", message, len(client.Requests))
	}
	if client.Requests[0].BaseURL != "https://simulator.example.com/v1" {
		t.Fatalf("base URL = %q", client.Requests[0].BaseURL)
	}
}

func TestResolveTargetIncludesDeploymentEndpoint(t *testing.T) {
	accountID := uuid.New()
	executionContext := repository.RunAgentExecutionContext{
		Deployment: repository.AgentDeploymentExecutionContext{
			ProviderAccount: &repository.ProviderAccountExecutionContext{
				ID:                  accountID,
				ProviderKey:         "custom",
				CredentialReference: "env://CUSTOM_API_KEY",
				BaseURL:             "https://simulator.example.com/v1",
			},
			ModelID: "controlled-model",
		},
	}
	providerKey, providerAccountID, credentialReference, baseURL, model, err := simulator.ResolveTarget(executionContext)
	if err != nil {
		t.Fatalf("ResolveTarget: %v", err)
	}
	if providerKey != "custom" || providerAccountID != accountID.String() || credentialReference != "env://CUSTOM_API_KEY" ||
		baseURL != "https://simulator.example.com/v1" || model != "controlled-model" {
		t.Fatalf("target = %q/%q/%q/%q/%q", providerKey, providerAccountID, credentialReference, baseURL, model)
	}
}

func TestTranscriptFromTurns_ClonesInput(t *testing.T) {
	t.Parallel()

	src := []simulator.TranscriptTurn{{Actor: "user", Content: "hello"}}
	cloned := simulator.TranscriptFromTurns(src)
	src[0].Content = "mutated"
	if cloned[0].Content != "hello" {
		t.Fatalf("TranscriptFromTurns() should clone; got %q", cloned[0].Content)
	}
}

func TestActorForEvent_MapsConversationActors(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		in, want string
	}{
		{"scripted", "user"},
		{"llm", "user"},
		{"human", "user"},
		{"assistant", "assistant"},
	} {
		if got := simulator.ActorForEvent(tc.in); got != tc.want {
			t.Fatalf("ActorForEvent(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
	if strings.TrimSpace(simulator.ActorForEvent("  human  ")) != "user" {
		t.Fatal("ActorForEvent should trim whitespace")
	}
}
