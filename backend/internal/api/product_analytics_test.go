package api

import (
	"context"

	"github.com/agentclash/agentclash/backend/internal/productanalytics"
)

type capturingProductAnalytics struct {
	events []productanalytics.Event
}

func (c *capturingProductAnalytics) Record(_ context.Context, event productanalytics.Event) {
	c.events = append(c.events, event)
}
