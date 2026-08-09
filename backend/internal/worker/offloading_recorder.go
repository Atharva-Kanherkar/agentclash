package worker

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"github.com/agentclash/agentclash/backend/internal/repository"
	"github.com/agentclash/agentclash/backend/internal/storage"
	"github.com/agentclash/agentclash/runtime/runevents"
	"github.com/google/uuid"
)

const runEventPayloadArtifactType = "run_event_payload"

// RunTenantLookup resolves org/workspace for artifact rows.
type RunTenantLookup interface {
	GetRunAnalyticsMetadata(ctx context.Context, runID uuid.UUID) (repository.RunAnalyticsMetadata, error)
	CreateArtifact(ctx context.Context, params repository.CreateArtifactParams) (repository.Artifact, error)
	MarkArtifactScheduledForDeletion(ctx context.Context, artifactID uuid.UUID) error
}

// OffloadingRecorder spills oversized run-event payloads to object storage
// before persisting a claim-check stub. When maxBytes <= 0, it is a passthrough.
type OffloadingRecorder struct {
	inner    RunEventRecorder
	lookup   RunTenantLookup
	store    storage.Store
	maxBytes int
	logger   *slog.Logger
}

var _ RunEventRecorder = (*OffloadingRecorder)(nil)

// NewOffloadingRecorder wraps inner. store may be nil only when maxBytes <= 0.
func NewOffloadingRecorder(
	inner RunEventRecorder,
	lookup RunTenantLookup,
	store storage.Store,
	maxBytes int,
	logger *slog.Logger,
) *OffloadingRecorder {
	if logger == nil {
		logger = slog.Default()
	}
	return &OffloadingRecorder{
		inner:    inner,
		lookup:   lookup,
		store:    store,
		maxBytes: maxBytes,
		logger:   logger,
	}
}

func (r *OffloadingRecorder) RecordRunEvent(ctx context.Context, params repository.RecordRunEventParams) (repository.RunEvent, error) {
	if r == nil || r.inner == nil {
		return repository.RunEvent{}, fmt.Errorf("offloading recorder is not configured")
	}
	if !runevents.ShouldOffload(params.Event.Payload, r.maxBytes) || r.store == nil {
		return r.inner.RecordRunEvent(ctx, params)
	}

	key := runevents.ObjectKeyForEvent(params.Event.RunAgentID, params.Event.EventID)
	body := params.Event.Payload
	meta, err := r.store.PutObject(ctx, storage.PutObjectInput{
		Key:         key,
		Body:        bytes.NewReader(body),
		SizeBytes:   int64(len(body)),
		ContentType: "application/json",
	})
	if err != nil {
		return repository.RunEvent{}, fmt.Errorf("put offloaded run event payload: %w", err)
	}

	tenant, err := r.lookup.GetRunAnalyticsMetadata(ctx, params.Event.RunID)
	if err != nil {
		_ = r.store.DeleteObject(ctx, key)
		return repository.RunEvent{}, fmt.Errorf("lookup run tenant for offload: %w", err)
	}

	size := int64(len(body))
	contentType := "application/json"
	artifact, err := r.lookup.CreateArtifact(ctx, repository.CreateArtifactParams{
		OrganizationID:  tenant.OrganizationID,
		WorkspaceID:     tenant.WorkspaceID,
		RunID:           &params.Event.RunID,
		RunAgentID:      &params.Event.RunAgentID,
		ArtifactType:    runEventPayloadArtifactType,
		StorageBucket:   meta.Bucket,
		StorageKey:      key,
		ContentType:     &contentType,
		SizeBytes:       &size,
		Visibility:      "private",
		RetentionStatus: "active",
	})
	if err != nil {
		_ = r.store.DeleteObject(ctx, key)
		return repository.RunEvent{}, fmt.Errorf("create run event payload artifact: %w", err)
	}

	compensate := func() {
		_ = r.store.DeleteObject(ctx, key)
		if markErr := r.lookup.MarkArtifactScheduledForDeletion(ctx, artifact.ID); markErr != nil {
			r.logger.Warn("failed to mark offload artifact for deletion",
				"artifact_id", artifact.ID, "storage_key", key, "error", markErr)
		}
	}

	stub, err := runevents.MarshalPayloadRef(key, len(body), params.Event.EventType)
	if err != nil {
		compensate()
		return repository.RunEvent{}, err
	}

	params.Event.Payload = stub
	params.ArtifactID = &artifact.ID
	event, err := r.inner.RecordRunEvent(ctx, params)
	if err != nil {
		compensate()
		return repository.RunEvent{}, err
	}
	return event, nil
}
