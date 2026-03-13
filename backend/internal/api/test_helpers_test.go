package api

import (
	"context"

	"github.com/Atharva-Kanherkar/agentclash/backend/internal/hostedruns"
	"github.com/google/uuid"
)

type stubHostedRunIngestionService struct{}

func (stubHostedRunIngestionService) IngestEvent(_ context.Context, _ uuid.UUID, _ string, _ hostedruns.Event) error {
	return nil
}
