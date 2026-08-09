package worker_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/agentclash/agentclash/backend/internal/repository"
	"github.com/agentclash/agentclash/backend/internal/storage"
	"github.com/agentclash/agentclash/backend/internal/worker"
	"github.com/agentclash/agentclash/runtime/runevents"
	"github.com/google/uuid"
)

type memStore struct {
	objects map[string][]byte
	bucket  string
}

func newMemStore() *memStore {
	return &memStore{objects: map[string][]byte{}, bucket: "test-bucket"}
}

func (s *memStore) Bucket() string { return s.bucket }

func (s *memStore) PutObject(_ context.Context, input storage.PutObjectInput) (storage.ObjectMetadata, error) {
	body, err := io.ReadAll(input.Body)
	if err != nil {
		return storage.ObjectMetadata{}, err
	}
	s.objects[input.Key] = body
	return storage.ObjectMetadata{Bucket: s.bucket, Key: input.Key, SizeBytes: int64(len(body)), ContentType: input.ContentType}, nil
}

func (s *memStore) OpenObject(_ context.Context, key string) (io.ReadCloser, storage.ObjectMetadata, error) {
	body, ok := s.objects[key]
	if !ok {
		return nil, storage.ObjectMetadata{}, storage.ErrObjectNotFound
	}
	return io.NopCloser(bytes.NewReader(body)), storage.ObjectMetadata{Bucket: s.bucket, Key: key, SizeBytes: int64(len(body))}, nil
}

func (s *memStore) DeleteObject(_ context.Context, key string) error {
	delete(s.objects, key)
	return nil
}

type stubRecorder struct {
	last repository.RecordRunEventParams
}

func (s *stubRecorder) RecordRunEvent(_ context.Context, params repository.RecordRunEventParams) (repository.RunEvent, error) {
	s.last = params
	return repository.RunEvent{
		RunID:          params.Event.RunID,
		RunAgentID:     params.Event.RunAgentID,
		SequenceNumber: 1,
		EventType:      params.Event.EventType,
		Source:         params.Event.Source,
		OccurredAt:     params.Event.OccurredAt,
		ArtifactID:     params.ArtifactID,
		Payload:        append([]byte(nil), params.Event.Payload...),
	}, nil
}

type stubLookup struct {
	meta              repository.RunAnalyticsMetadata
	artifacts         map[uuid.UUID]repository.Artifact
	markedForDeletion []uuid.UUID
}

func (s *stubLookup) GetRunAnalyticsMetadata(context.Context, uuid.UUID) (repository.RunAnalyticsMetadata, error) {
	return s.meta, nil
}

func (s *stubLookup) CreateArtifact(_ context.Context, params repository.CreateArtifactParams) (repository.Artifact, error) {
	if s.artifacts == nil {
		s.artifacts = map[uuid.UUID]repository.Artifact{}
	}
	art := repository.Artifact{
		ID:              uuid.New(),
		OrganizationID:  params.OrganizationID,
		WorkspaceID:     params.WorkspaceID,
		StorageBucket:   params.StorageBucket,
		StorageKey:      params.StorageKey,
		ArtifactType:    params.ArtifactType,
		RetentionStatus: params.RetentionStatus,
	}
	s.artifacts[art.ID] = art
	return art, nil
}

func (s *stubLookup) MarkArtifactScheduledForDeletion(_ context.Context, artifactID uuid.UUID) error {
	s.markedForDeletion = append(s.markedForDeletion, artifactID)
	if art, ok := s.artifacts[artifactID]; ok {
		art.RetentionStatus = "scheduled_for_deletion"
		s.artifacts[artifactID] = art
	}
	return nil
}

type failingRecorder struct {
	err error
}

func (f failingRecorder) RecordRunEvent(context.Context, repository.RecordRunEventParams) (repository.RunEvent, error) {
	return repository.RunEvent{}, f.err
}

func TestOffloadingRecorder_SpillsLargePayload(t *testing.T) {
	store := newMemStore()
	inner := &stubRecorder{}
	runID := uuid.New()
	agentID := uuid.New()
	lookup := &stubLookup{meta: repository.RunAnalyticsMetadata{
		RunID: runID, WorkspaceID: uuid.New(), OrganizationID: uuid.New(),
	}}
	rec := worker.NewOffloadingRecorder(inner, lookup, store, 32, slog.Default())

	payload, _ := json.Marshal(map[string]string{"blob": string(bytes.Repeat([]byte("x"), 64))})
	event, err := rec.RecordRunEvent(context.Background(), repository.RecordRunEventParams{
		Event: runevents.Envelope{
			EventID:       "e1",
			SchemaVersion: runevents.SchemaVersionV1,
			RunID:         runID,
			RunAgentID:    agentID,
			EventType:     runevents.EventTypeModelOutputDelta,
			Source:        runevents.SourceNativeEngine,
			OccurredAt:    time.Now().UTC(),
			Payload:       payload,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ref, ok := runevents.ParsePayloadRef(event.Payload)
	if !ok {
		t.Fatalf("expected stub payload, got %s", event.Payload)
	}
	if _, ok := store.objects[ref.Ref]; !ok {
		t.Fatalf("object missing for %s", ref.Ref)
	}
	if event.ArtifactID == nil {
		t.Fatal("expected artifact id")
	}
}

func TestOffloadingRecorder_DisabledPassthrough(t *testing.T) {
	store := newMemStore()
	inner := &stubRecorder{}
	rec := worker.NewOffloadingRecorder(inner, &stubLookup{}, store, 0, slog.Default())
	payload := json.RawMessage(`{"ok":true}`)
	event, err := rec.RecordRunEvent(context.Background(), repository.RecordRunEventParams{
		Event: runevents.Envelope{
			EventID:       "e1",
			SchemaVersion: runevents.SchemaVersionV1,
			RunID:         uuid.New(),
			RunAgentID:    uuid.New(),
			EventType:     runevents.EventTypeModelOutputDelta,
			Source:        runevents.SourceNativeEngine,
			OccurredAt:    time.Now().UTC(),
			Payload:       payload,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(event.Payload) != string(payload) {
		t.Fatalf("payload = %s", event.Payload)
	}
	if len(store.objects) != 0 {
		t.Fatal("store should be empty when disabled")
	}
}

func TestOffloadingRecorder_RollbackArtifactOnInnerFailure(t *testing.T) {
	store := newMemStore()
	runID := uuid.New()
	agentID := uuid.New()
	lookup := &stubLookup{meta: repository.RunAnalyticsMetadata{
		RunID: runID, WorkspaceID: uuid.New(), OrganizationID: uuid.New(),
	}}
	rec := worker.NewOffloadingRecorder(failingRecorder{err: errors.New("persist failed")}, lookup, store, 32, slog.Default())

	payload, _ := json.Marshal(map[string]string{"blob": string(bytes.Repeat([]byte("x"), 64))})
	_, err := rec.RecordRunEvent(context.Background(), repository.RecordRunEventParams{
		Event: runevents.Envelope{
			EventID:       "e1",
			SchemaVersion: runevents.SchemaVersionV1,
			RunID:         runID,
			RunAgentID:    agentID,
			EventType:     runevents.EventTypeModelOutputDelta,
			Source:        runevents.SourceNativeEngine,
			OccurredAt:    time.Now().UTC(),
			Payload:       payload,
		},
	})
	if err == nil {
		t.Fatal("expected persist failure")
	}
	if len(store.objects) != 0 {
		t.Fatalf("expected object deleted, still have %d", len(store.objects))
	}
	if len(lookup.markedForDeletion) != 1 {
		t.Fatalf("markedForDeletion = %v, want 1 artifact", lookup.markedForDeletion)
	}
	art := lookup.artifacts[lookup.markedForDeletion[0]]
	if art.RetentionStatus != "scheduled_for_deletion" {
		t.Fatalf("retention = %q, want scheduled_for_deletion", art.RetentionStatus)
	}
}
