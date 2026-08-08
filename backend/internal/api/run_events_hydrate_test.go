package api

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/agentclash/agentclash/runtime/runevents"
)

func TestHydrateLiveRunEventData(t *testing.T) {
	body := `{"delta":"hello-world"}`
	open := runevents.OpenFunc(func(_ context.Context, key string) (io.ReadCloser, error) {
		if key != "run-events/a/b.json" {
			t.Fatalf("key = %s", key)
		}
		return io.NopCloser(strings.NewReader(body)), nil
	})
	resolver := runevents.NewResolver(open, 8)
	stub, err := runevents.MarshalPayloadRef("run-events/a/b.json", len(body), runevents.EventTypeModelOutputDelta)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(runevents.Envelope{
		EventID:       "e1",
		SchemaVersion: runevents.SchemaVersionV1,
		EventType:     runevents.EventTypeModelOutputDelta,
		Payload:       stub,
	})
	out, err := hydrateLiveRunEventData(context.Background(), resolver, raw)
	if err != nil {
		t.Fatal(err)
	}
	var got runevents.Envelope
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if string(got.Payload) != body {
		t.Fatalf("payload = %s", got.Payload)
	}
}
