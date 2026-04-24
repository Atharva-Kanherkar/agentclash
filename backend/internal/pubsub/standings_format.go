package pubsub

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FormatStandingsInput is the pure-function input for the race-context
// newswire formatter. Passing Now explicitly keeps the function
// deterministic under test.
type FormatStandingsInput struct {
	Snapshot       map[uuid.UUID]StandingsEntry
	SelfRunAgentID uuid.UUID
	SelfStepIndex  int
	Now            time.Time
}

// FormatStandings renders a race-context newswire message. It returns the
// text that will be injected as a `role=user` message and an estimated
// token count for billable-token accounting (see slice 8). The formatter
// is deliberately neutral and factual — no directives, no second-person
// imperatives — so the injection does not prime the model toward any
// behavior beyond observing peer progress.
//
// Ordering: submitters pinned to top in submission order, remaining peers
// ranked by step descending, ties broken by run_agent_id for determinism.
func FormatStandings(in FormatStandingsInput) (string, int) {
	entries := orderStandings(in.Snapshot)

	running, submitted := 0, 0
	for _, e := range entries {
		switch e.State {
		case StandingsStateSubmitted:
			submitted++
		case StandingsStateRunning, StandingsStateNotStarted, "":
			running++
		}
		// FAILED and TIMED OUT are shown but not counted in either total.
	}

	var b strings.Builder
	fmt.Fprintf(&b, "[RACE UPDATE · your step %d]\n", in.SelfStepIndex)
	fmt.Fprintf(&b, "%d agents running, %d submitted.\n", running, submitted)

	for _, entry := range entries {
		b.WriteString("- ")
		b.WriteString(formatEntryLine(entry, in.SelfRunAgentID, in.Now))
		b.WriteString("\n")
	}

	text := strings.TrimRight(b.String(), "\n")
	return text, estimateTokens(text)
}

func orderStandings(snapshot map[uuid.UUID]StandingsEntry) []StandingsEntry {
	entries := make([]StandingsEntry, 0, len(snapshot))
	for _, e := range snapshot {
		entries = append(entries, e)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		// Submitters float to the top, ordered by submission time.
		iSubmitted := entries[i].State == StandingsStateSubmitted
		jSubmitted := entries[j].State == StandingsStateSubmitted
		if iSubmitted != jSubmitted {
			return iSubmitted
		}
		if iSubmitted && jSubmitted {
			// Earlier submission shows first.
			switch {
			case entries[i].SubmittedAt == nil && entries[j].SubmittedAt == nil:
				// fall through to step-based tiebreak
			case entries[i].SubmittedAt == nil:
				return false
			case entries[j].SubmittedAt == nil:
				return true
			default:
				if !entries[i].SubmittedAt.Equal(*entries[j].SubmittedAt) {
					return entries[i].SubmittedAt.Before(*entries[j].SubmittedAt)
				}
			}
		}
		// Higher step first.
		if entries[i].Step != entries[j].Step {
			return entries[i].Step > entries[j].Step
		}
		// Deterministic tiebreak.
		return entries[i].RunAgentID.String() < entries[j].RunAgentID.String()
	})
	return entries
}

func formatEntryLine(entry StandingsEntry, selfRunAgentID uuid.UUID, now time.Time) string {
	modelLabel := entry.Model
	if modelLabel == "" {
		// Fall back to a short id so the model has something to anchor on
		// before the first model.call.completed populates the name.
		modelLabel = "agent-" + entry.RunAgentID.String()[:8]
	}
	prefix := modelLabel
	if entry.RunAgentID == selfRunAgentID {
		prefix = "you (" + modelLabel + ")"
	}

	switch entry.State {
	case StandingsStateSubmitted:
		elapsed := formatElapsed(entry.StartedAt, entry.SubmittedAt)
		return fmt.Sprintf("%s — submitted at step %d (%s elapsed) · verifying", prefix, entry.Step, elapsed)
	case StandingsStateFailed:
		return fmt.Sprintf("%s — FAILED at step %d", prefix, entry.Step)
	case StandingsStateTimedOut:
		return fmt.Sprintf("%s — TIMED OUT at step %d", prefix, entry.Step)
	case StandingsStateNotStarted, "":
		return fmt.Sprintf("%s — not started", prefix)
	default:
		// Running (or any unexpected state): show progress.
		return fmt.Sprintf("%s — %d steps, %d tool calls, %d tokens", prefix, entry.Step, entry.ToolCalls, entry.TokensUsed)
	}
}

func formatElapsed(start, end *time.Time) string {
	if start == nil || end == nil {
		return "—"
	}
	d := end.Sub(*start)
	if d < 0 {
		d = 0
	}
	minutes := int(d / time.Minute)
	seconds := int((d % time.Minute) / time.Second)
	return fmt.Sprintf("%dm%02ds", minutes, seconds)
}

// estimateTokens is a cheap approximation of the prompt-side token count
// (roughly 4 chars per token for English prose). Used solely for the
// race-context token-accounting split in slice 8 — we don't need
// provider-accurate tokenization for spend observability.
func estimateTokens(text string) int {
	if text == "" {
		return 0
	}
	// Round up so a 1-char string still reports at least 1 token.
	return (len(text) + 3) / 4
}
