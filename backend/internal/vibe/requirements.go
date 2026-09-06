package vibe

import (
	"github.com/google/uuid"
	"strings"
)

// DecideRequirement runs only inside the authenticated, revision-checked edit
// transaction. Neither authoring output nor conversation summaries can call it.
func DecideRequirement(v *Session, id uuid.UUID, status string, replacement *string) error {
	for i := range v.Document.Requirements {
		q := &v.Document.Requirements[i]
		if q.ID != id {
			continue
		}
		if q.Status == "proposed" && replacement == nil && (status == "accepted" || status == "rejected") {
			q.Status = status
			if status == "accepted" {
				now := timestamp()
				q.AcceptedBy, q.AcceptedAt = v.Actor, &now
			}
			return nil
		}
		if q.Status == "accepted" && status == "superseded" && replacement != nil {
			if strings.TrimSpace(*replacement) == "" || len(*replacement) > 4096 {
				return fault("invalid_requirement", "Write a replacement requirement of at most 4096 bytes.")
			}
			now, source, next := timestamp(), uuid.New(), uuid.New()
			q.Status = "superseded"
			v.Document.Messages = append(v.Document.Messages, Message{source, "user", "Replace the confirmed requirement with: " + *replacement, now})
			v.Document.Requirements = append(v.Document.Requirements, Requirement{ID: next, Statement: *replacement, Status: "accepted", SourceMessageID: source, ProposedBy: "user", AcceptedBy: v.Actor, AcceptedAt: &now, SupersedesID: &id})
			return nil
		}
		return fault("invalid_requirement", "Only a proposal can be confirmed or dismissed. Confirmed requirements need an explicit replacement.")
	}
	return fault("not_found", "Requirement is unavailable.")
}
