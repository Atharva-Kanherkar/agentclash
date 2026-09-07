package vibe

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/agentclash/agentclash/runtime/provider"
	"github.com/google/uuid"
)

type repairCompiler struct{}

func (repairCompiler) ValidateDraft(json.RawMessage, Limits) error { return nil }

func (repairCompiler) Instructions() string {
	return "Create three cases grounded in the supplied policy."
}
func (repairCompiler) Compile(json.RawMessage, string, uuid.UUID, Limits) (Compiled, error) {
	return Compiled{}, nil
}

func TestIntegrationVibeFreeAuthoringRepair(t *testing.T) {
	for _, mode := range []string{"corrected", "still_invalid", "question"} {
		t.Run(mode, func(t *testing.T) {
			s := integrationStore(t)
			v := anonSession(t, s)
			cfg := freeConfig()
			gate := testGate(t)
			ctx := context.Background()
			blueprint := json.RawMessage(`{"cases":[{"key":"eligible"},{"key":"late"},{"key":"missing-date"}]}`)
			valid := string(raw(map[string]any{"reply": "Review the three examples.", "requirements": []string{}, "draft": map[string]any{"title": "Refund support", "agent_prompt": "Allow refunds within 30 days. Ask for a missing date.", "blueprint": blueprint}}))
			// Like the live failure: a model echoes a large prompt metadata field.
			// Put it last so partial decoding has already populated the draft.
			invalid := strings.TrimSuffix(valid, "}") + `,"prompt_metadata":"` + strings.Repeat("metadata", 1500) + `"}`
			calls := 0
			fake := callFunc(func(_ context.Context, request provider.Request) (provider.Response, error) {
				calls++
				if _, err := CountContext(request, cfg.Profiles[cfg.DefaultModel], LimitsFor(true)); err != nil {
					t.Fatal("oversized request reached provider", err)
				}
				output := invalid
				if calls > 1 {
					if strings.Contains(request.Messages[len(request.Messages)-1].Content, "invalid_response") || !strings.Contains(request.Messages[1].Content, "within 30 days") {
						t.Fatal("repair did not retain original intent and omit oversized invalid output")
					}
					switch mode {
					case "corrected":
						output = valid
					case "question":
						output = `{"reply":"What should happen if the date is missing?","requirements":[]}`
					}
				}
				zero := json.Number("0")
				return provider.Response{OutputText: output, Usage: provider.Usage{InputTokens: 100, OutputTokens: 200, CostUSD: &zero}}, nil
			})
			svc := &Service{Store: s, Config: cfg, Gate: gate, Compiler: repairCompiler{}}
			runner := Runner{Service: svc, Gateway: &Gateway{Store: s, Config: cfg, Gate: gate, Client: fake}}
			o, err := svc.Prepare(ctx, v.Actor, v.ID, Submission{ClientID: uuid.New(), Models: cfg.DefaultModels(), Kind: "message", Content: "Build a support agent: refunds within 30 days, decline late requests, ask for missing dates. Create three cases."})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				_ = s.Finish(ctx, o.ID, &Fault{"test_cleanup", "Test ended."})
				_, _ = s.DB.Exec(ctx, "DELETE FROM vibe_attempts WHERE operation_id=$1", o.ID)
			})
			err = runner.Execute(ctx, o.ID)
			if mode == "still_invalid" {
				var f *Fault
				if !errors.As(err, &f) || f.Code != "invalid_draft" {
					t.Fatalf("unexpected correction failure: %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if calls != 2 {
				t.Fatalf("correction escaped its two-call bound: %d", calls)
			}
			if err = s.Finish(ctx, o.ID, issueFrom(err)); err != nil {
				t.Fatal(err)
			}
			current, err := s.GetSession(ctx, v.Actor, v.ID)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "corrected" {
				if len(current.Document.Artifacts) != 1 {
					t.Fatal("correction did not produce exactly one draft")
				}
				var got, want any
				if json.Unmarshal(current.Document.Artifacts[0].Blueprint, &got) != nil || json.Unmarshal(blueprint, &want) != nil || !reflect.DeepEqual(got, want) {
					t.Fatal("correction lost requested coverage")
				}
			} else if len(current.Document.Artifacts) != 0 {
				t.Fatal("invalid first-response draft leaked into the final document")
			}
			finished, err := s.Operation(ctx, o.ID)
			if err != nil || finished.ActualCost == nil || *finished.ActualCost != 0 || finished.Billing != Released {
				t.Fatalf("zero-cost accounting was lost: %+v %v", finished, err)
			}
		})
	}
}
