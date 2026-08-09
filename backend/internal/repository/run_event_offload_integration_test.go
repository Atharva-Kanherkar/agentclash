package repository_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/agentclash/agentclash/backend/internal/repository"
	"github.com/agentclash/agentclash/backend/internal/storage"
	"github.com/agentclash/agentclash/backend/internal/worker"
	"github.com/agentclash/agentclash/runtime/runevents"
	"github.com/google/uuid"
	"log/slog"
)

func TestRunEventOffload_SpillHydrateExportAndPurge(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	fixture := seedFixture(t, ctx, db)

	root := t.TempDir()
	store, err := storage.NewFilesystemStore(storage.Config{
		Backend:        storage.BackendFilesystem,
		Bucket:         "test-events",
		FilesystemRoot: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	resolver := runevents.NewResolver(runevents.OpenFunc(func(ctx context.Context, key string) (io.ReadCloser, error) {
		rc, _, err := store.OpenObject(ctx, key)
		return rc, err
	}), 8)
	repo := repository.New(db).WithPayloadResolver(resolver)
	recorder := worker.NewOffloadingRecorder(repo, repo, store, 64, slog.Default())

	payloadObj := map[string]any{"blob": string(bytes.Repeat([]byte("Z"), 200))}
	payload, _ := json.Marshal(payloadObj)

	eventID := "offload-" + uuid.NewString()
	recorded, err := recorder.RecordRunEvent(ctx, repository.RecordRunEventParams{
		Event: runevents.Envelope{
			EventID:       eventID,
			SchemaVersion: runevents.SchemaVersionV1,
			RunID:         fixture.runID,
			RunAgentID:    fixture.primaryRunAgentID,
			EventType:     runevents.EventTypeModelOutputDelta,
			Source:        runevents.SourceNativeEngine,
			OccurredAt:    time.Now().UTC(),
			Payload:       payload,
		},
	})
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	if _, ok := runevents.ParsePayloadRef(recorded.Payload); !ok {
		// Recorded return may already be stub from offloader; check DB raw without hydrate
		t.Fatalf("offloader should return stub, got %s", recorded.Payload)
	}

	// Hydrated list must match original payload bytes.
	listed, err := repo.ListRunEventsByRunAgentID(ctx, fixture.primaryRunAgentID)
	if err != nil {
		t.Fatal(err)
	}
	var found *repository.RunEvent
	for i := range listed {
		if listed[i].SequenceNumber == recorded.SequenceNumber {
			found = &listed[i]
			break
		}
	}
	if found == nil {
		t.Fatal("event missing from list")
	}
	if !bytes.Equal(found.Payload, payload) {
		t.Fatalf("hydrated payload mismatch:\n got %s\nwant %s", found.Payload, payload)
	}

	// Object must exist on disk.
	ref, _ := runevents.ParsePayloadRef(recorded.Payload)
	objPath := filepath.Join(root, "test-events", filepath.FromSlash(ref.Ref))
	if _, err := os.Stat(objPath); err != nil {
		t.Fatalf("object missing at %s: %v", objPath, err)
	}

	if err := repo.HardDeleteRun(ctx, store, fixture.runID); err != nil {
		t.Fatalf("hard delete: %v", err)
	}
	if _, err := os.Stat(objPath); !os.IsNotExist(err) {
		t.Fatalf("expected object deleted, stat err=%v", err)
	}
}
