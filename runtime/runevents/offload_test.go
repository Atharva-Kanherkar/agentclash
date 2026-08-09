package runevents_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/agentclash/agentclash/runtime/runevents"
	"github.com/google/uuid"
)

func TestPayloadRef_RoundTrip(t *testing.T) {
	raw, err := runevents.MarshalPayloadRef("run-events/a/b.json", 1024, runevents.EventTypeModelOutputDelta)
	if err != nil {
		t.Fatal(err)
	}
	ref, ok := runevents.ParsePayloadRef(raw)
	if !ok {
		t.Fatalf("expected stub, got %s", raw)
	}
	if ref.Ref != "run-events/a/b.json" || ref.Bytes != 1024 || ref.Type != string(runevents.EventTypeModelOutputDelta) {
		t.Fatalf("ref = %#v", ref)
	}
	if _, ok := runevents.ParsePayloadRef(json.RawMessage(`{"delta":"hi"}`)); ok {
		t.Fatal("inline payload should not parse as stub")
	}
}

func TestParsePayloadRef_RejectsUnsafeOrMalformed(t *testing.T) {
	cases := []string{
		`{"$ref":"../../secrets","bytes":1,"type":"x"}`,
		`{"$ref":"other/bucket/key.json","bytes":1,"type":"x"}`,
		`{"$ref":"run-events/a/b.json","bytes":1,"type":"x","extra":true}`,
		`{"$ref":"run-events/a/b.json","bytes":0,"type":"x"}`,
		`{"$ref":"run-events/a/b","bytes":1,"type":"x"}`,
		`{"$ref":"run-events/../escape.json","bytes":1,"type":"x"}`,
		`{"$ref":"run-events/a/b.json","type":"x"}`,
	}
	for _, raw := range cases {
		if _, ok := runevents.ParsePayloadRef(json.RawMessage(raw)); ok {
			t.Fatalf("expected reject for %s", raw)
		}
	}
}

func TestShouldOffload(t *testing.T) {
	if runevents.ShouldOffload([]byte("x"), 0) {
		t.Fatal("maxBytes=0 must disable")
	}
	if runevents.ShouldOffload(bytes.Repeat([]byte("a"), 10), 32) {
		t.Fatal("under threshold")
	}
	if !runevents.ShouldOffload(bytes.Repeat([]byte("a"), 33), 32) {
		t.Fatal("over threshold")
	}
}

func TestObjectKeyForEvent(t *testing.T) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	got := runevents.ObjectKeyForEvent(id, "evt-1")
	want := "run-events/11111111-1111-1111-1111-111111111111/evt-1.json"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolver_LRU(t *testing.T) {
	opens := 0
	bodies := map[string]string{
		"run-events/a/k1.json": `{"n":1}`,
		"run-events/a/k2.json": `{"n":2}`,
		"run-events/a/k3.json": `{"n":3}`,
	}
	open := runevents.OpenFunc(func(_ context.Context, key string) (io.ReadCloser, error) {
		opens++
		return io.NopCloser(strings.NewReader(bodies[key])), nil
	})
	r := runevents.NewResolver(open, 2)

	stub := func(key string) json.RawMessage {
		raw, _ := runevents.MarshalPayloadRef(key, 10, runevents.EventTypeModelOutputDelta)
		return raw
	}
	ctx := context.Background()
	if _, err := r.Resolve(ctx, stub("run-events/a/k1.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Resolve(ctx, stub("run-events/a/k1.json")); err != nil {
		t.Fatal(err)
	}
	if opens != 1 {
		t.Fatalf("opens = %d, want 1 after cache hit", opens)
	}
	if _, err := r.Resolve(ctx, stub("run-events/a/k2.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Resolve(ctx, stub("run-events/a/k3.json")); err != nil {
		t.Fatal(err)
	}
	// k1 should be evicted
	if _, err := r.Resolve(ctx, stub("run-events/a/k1.json")); err != nil {
		t.Fatal(err)
	}
	if opens < 4 {
		t.Fatalf("opens = %d, want >=4 after eviction", opens)
	}
}
