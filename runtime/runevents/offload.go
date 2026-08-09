package runevents

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// PayloadRefMarker is the JSON key that identifies an offloaded payload stub.
const PayloadRefMarker = "$ref"

// PayloadRef is the stub stored inline in run_events.payload when the full
// body has been spilled to object storage (claim-check pattern).
type PayloadRef struct {
	Ref   string `json:"$ref"`
	Bytes int    `json:"bytes"`
	Type  string `json:"type"`
}

// MarshalPayloadRef returns the stub JSON for an offloaded payload.
func MarshalPayloadRef(ref string, sizeBytes int, eventType Type) (json.RawMessage, error) {
	if ref == "" {
		return nil, fmt.Errorf("payload ref key is required")
	}
	return json.Marshal(PayloadRef{
		Ref:   ref,
		Bytes: sizeBytes,
		Type:  string(eventType),
	})
}

// ParsePayloadRef returns (ref, true) when payload is an offload stub.
// Only the exact stub shape ({"$ref","bytes","type"}) is accepted, and the
// storage key must be under run-events/ ending in .json.
func ParsePayloadRef(payload json.RawMessage) (PayloadRef, bool) {
	if len(payload) == 0 {
		return PayloadRef{}, false
	}
	var ref PayloadRef
	if err := json.Unmarshal(payload, &ref); err != nil || ref.Ref == "" || ref.Bytes <= 0 || ref.Type == "" {
		return PayloadRef{}, false
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil || len(raw) != 3 {
		return PayloadRef{}, false
	}
	for _, key := range []string{PayloadRefMarker, "bytes", "type"} {
		if _, ok := raw[key]; !ok {
			return PayloadRef{}, false
		}
	}
	if !strings.HasPrefix(ref.Ref, "run-events/") || !strings.HasSuffix(ref.Ref, ".json") {
		return PayloadRef{}, false
	}
	if strings.Contains(ref.Ref, "..") {
		return PayloadRef{}, false
	}
	return ref, true
}

// ObjectKeyForEvent builds the storage key for an offloaded run-event payload.
// Uses event_id (not sequence) so the object can be written before the DB
// assigns sequence_number under concurrent fan-out.
func ObjectKeyForEvent(runAgentID uuid.UUID, eventID string) string {
	return fmt.Sprintf("run-events/%s/%s.json", runAgentID.String(), eventID)
}

// ShouldOffload reports whether a payload should be spilled.
// maxBytes <= 0 disables offloading (today's inline behavior).
func ShouldOffload(payload []byte, maxBytes int) bool {
	if maxBytes <= 0 {
		return false
	}
	return len(payload) > maxBytes
}

// ObjectOpener opens objects by storage key.
type ObjectOpener interface {
	OpenObject(ctx context.Context, key string) (io.ReadCloser, error)
}

// OpenFunc adapts a function to ObjectOpener.
type OpenFunc func(ctx context.Context, key string) (io.ReadCloser, error)

func (f OpenFunc) OpenObject(ctx context.Context, key string) (io.ReadCloser, error) {
	return f(ctx, key)
}

// Resolver hydrates offloaded payloads. Hot keys are cached in an LRU.
type Resolver struct {
	open   ObjectOpener
	mu     sync.Mutex
	lru    map[string]json.RawMessage
	order  []string
	maxLRU int
}

// NewResolver constructs a resolver with a small in-process LRU (default 64).
func NewResolver(open ObjectOpener, maxLRU int) *Resolver {
	if maxLRU <= 0 {
		maxLRU = 64
	}
	return &Resolver{
		open:   open,
		lru:    make(map[string]json.RawMessage, maxLRU),
		maxLRU: maxLRU,
	}
}

// Resolve returns the original payload bytes. Non-stub payloads pass through.
func (r *Resolver) Resolve(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	if r == nil || r.open == nil {
		return payload, nil
	}
	ref, ok := ParsePayloadRef(payload)
	if !ok {
		return payload, nil
	}
	r.mu.Lock()
	if cached, hit := r.lru[ref.Ref]; hit {
		r.mu.Unlock()
		return append(json.RawMessage(nil), cached...), nil
	}
	r.mu.Unlock()

	rc, err := r.open.OpenObject(ctx, ref.Ref)
	if err != nil {
		return nil, fmt.Errorf("open offloaded run event payload %q: %w", ref.Ref, err)
	}
	defer rc.Close()
	body, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("read offloaded run event payload %q: %w", ref.Ref, err)
	}
	if !json.Valid(body) {
		return nil, fmt.Errorf("offloaded run event payload %q is not valid JSON", ref.Ref)
	}
	out := json.RawMessage(body)
	r.remember(ref.Ref, out)
	return append(json.RawMessage(nil), out...), nil
}

func (r *Resolver) remember(key string, value json.RawMessage) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.lru[key]; ok {
		r.lru[key] = append(json.RawMessage(nil), value...)
		return
	}
	if len(r.order) >= r.maxLRU {
		evict := r.order[0]
		r.order = r.order[1:]
		delete(r.lru, evict)
	}
	r.order = append(r.order, key)
	r.lru[key] = append(json.RawMessage(nil), value...)
}
