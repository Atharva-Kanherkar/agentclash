package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// StandingsEntry is the per-agent snapshot stored in the Redis hash
// `run:{run_id}:standings` under the field `agent:{run_agent_id}`. See
// issue #400 for design. Fields are optional; the StandingsWriter merges
// partial updates in with HSET/HGET.
type StandingsEntry struct {
	RunAgentID   uuid.UUID          `json:"run_agent_id"`
	Model        string             `json:"model,omitempty"`
	Step         int                `json:"step"`
	ToolCalls    int                `json:"tool_calls"`
	TokensUsed   int64              `json:"tokens_used"`
	State        StandingsState     `json:"state"`
	SubmittedAt  *time.Time         `json:"submitted_at,omitempty"`
	FailedAt     *time.Time         `json:"failed_at,omitempty"`
	StartedAt    *time.Time         `json:"started_at,omitempty"`
	LastEventAt  time.Time          `json:"last_event_at"`
}

// StandingsState matches the states documented in the race-context issue:
// not_started, running, submitted, failed, timed_out.
type StandingsState string

const (
	StandingsStateNotStarted StandingsState = "not_started"
	StandingsStateRunning    StandingsState = "running"
	StandingsStateSubmitted  StandingsState = "submitted"
	StandingsStateFailed     StandingsState = "failed"
	StandingsStateTimedOut   StandingsState = "timed_out"
)

// StandingsStore is the per-run Redis hash for agent standings. All methods
// must be safe to call from concurrent goroutines (the executor writes
// events from multiple agents in parallel).
type StandingsStore interface {
	// Update applies a partial update for one agent. Fields in `updates`
	// overwrite existing values; zero/empty fields are ignored. The store
	// is responsible for merging, so callers can send only the fields that
	// changed.
	Update(ctx context.Context, runID uuid.UUID, updates StandingsEntry) error
	// Snapshot returns the current standings for all agents in a run.
	// Returns an empty map if no standings are recorded. Callers must not
	// rely on strict consistency: peer agents may be mid-update.
	Snapshot(ctx context.Context, runID uuid.UUID) (map[uuid.UUID]StandingsEntry, error)
	// Close releases any underlying resources.
	Close() error
}

// StandingsHashKey returns the Redis hash key for a run's standings.
func StandingsHashKey(runID uuid.UUID) string {
	return "run:" + runID.String() + ":standings"
}

// StandingsField returns the hash field name for a given agent.
func StandingsField(runAgentID uuid.UUID) string {
	return "agent:" + runAgentID.String()
}

// standingsTTL matches the issue spec: 1 hour after the last update, the
// hash expires. Reruns past this window repopulate naturally.
const standingsTTL = time.Hour

// RedisStandingsStore persists standings as JSON in a Redis hash, one
// field per agent. Updates are merged: the store reads the existing field,
// applies non-zero fields from the update, and writes the merged value.
type RedisStandingsStore struct {
	client *redis.Client
}

var _ StandingsStore = (*RedisStandingsStore)(nil)

// NewRedisStandingsStore returns a store backed by the given client. The
// caller retains ownership of the client — Close is a no-op here so the
// client outlives the store.
func NewRedisStandingsStore(client *redis.Client) *RedisStandingsStore {
	return &RedisStandingsStore{client: client}
}

func (s *RedisStandingsStore) Update(ctx context.Context, runID uuid.UUID, updates StandingsEntry) error {
	if updates.RunAgentID == uuid.Nil {
		return fmt.Errorf("standings update requires run_agent_id")
	}
	key := StandingsHashKey(runID)
	field := StandingsField(updates.RunAgentID)

	existing, err := s.fetchEntry(ctx, key, field)
	if err != nil {
		return err
	}
	merged := mergeEntry(existing, updates)
	merged.LastEventAt = time.Now().UTC()

	payload, err := json.Marshal(merged)
	if err != nil {
		return fmt.Errorf("marshal standings entry: %w", err)
	}

	pipe := s.client.TxPipeline()
	pipe.HSet(ctx, key, field, payload)
	pipe.Expire(ctx, key, standingsTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("redis hset standings: %w", err)
	}
	return nil
}

func (s *RedisStandingsStore) Snapshot(ctx context.Context, runID uuid.UUID) (map[uuid.UUID]StandingsEntry, error) {
	key := StandingsHashKey(runID)
	raw, err := s.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis hgetall standings: %w", err)
	}
	out := make(map[uuid.UUID]StandingsEntry, len(raw))
	for field, value := range raw {
		var entry StandingsEntry
		if err := json.Unmarshal([]byte(value), &entry); err != nil {
			// Skip malformed entries rather than failing the snapshot.
			continue
		}
		if entry.RunAgentID == uuid.Nil {
			// Field-encoded agent id is the canonical source.
			if id, parseErr := uuid.Parse(field[len("agent:"):]); parseErr == nil {
				entry.RunAgentID = id
			}
		}
		out[entry.RunAgentID] = entry
	}
	return out, nil
}

func (s *RedisStandingsStore) Close() error { return nil }

func (s *RedisStandingsStore) fetchEntry(ctx context.Context, key, field string) (StandingsEntry, error) {
	raw, err := s.client.HGet(ctx, key, field).Result()
	if err != nil {
		if err == redis.Nil {
			return StandingsEntry{}, nil
		}
		return StandingsEntry{}, fmt.Errorf("redis hget standings: %w", err)
	}
	var entry StandingsEntry
	if err := json.Unmarshal([]byte(raw), &entry); err != nil {
		// Treat corrupt state as missing so subsequent updates overwrite.
		return StandingsEntry{}, nil
	}
	return entry, nil
}

// mergeEntry applies non-zero fields from `updates` onto `existing`. Scalar
// zero values in `updates` are treated as "don't touch" so callers can send
// partial updates (e.g. only step change, only token delta).
func mergeEntry(existing, updates StandingsEntry) StandingsEntry {
	if updates.RunAgentID != uuid.Nil {
		existing.RunAgentID = updates.RunAgentID
	}
	if updates.Model != "" {
		existing.Model = updates.Model
	}
	if updates.Step > existing.Step {
		existing.Step = updates.Step
	}
	existing.ToolCalls += updates.ToolCalls
	existing.TokensUsed += updates.TokensUsed
	if updates.State != "" {
		existing.State = updates.State
	}
	if updates.SubmittedAt != nil {
		existing.SubmittedAt = updates.SubmittedAt
	}
	if updates.FailedAt != nil {
		existing.FailedAt = updates.FailedAt
	}
	if updates.StartedAt != nil && existing.StartedAt == nil {
		existing.StartedAt = updates.StartedAt
	}
	return existing
}

// NoopStandingsStore is used when Redis is not configured. All operations
// are no-ops and Snapshot returns an empty map. Keeping this type keeps the
// worker wiring uniform regardless of deployment posture.
type NoopStandingsStore struct{}

var _ StandingsStore = NoopStandingsStore{}

func (NoopStandingsStore) Update(context.Context, uuid.UUID, StandingsEntry) error { return nil }
func (NoopStandingsStore) Snapshot(context.Context, uuid.UUID) (map[uuid.UUID]StandingsEntry, error) {
	return map[uuid.UUID]StandingsEntry{}, nil
}
func (NoopStandingsStore) Close() error { return nil }
