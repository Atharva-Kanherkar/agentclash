package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/agentclash/agentclash/cli/internal/output"
	"github.com/spf13/cobra"
)

type evalsetLogLine struct {
	MatrixKey string
	RunID     string
	Event     string
	ID        string
	Data      []byte
	Err       error
	Done      bool
}

func runEvalsetLogs(cmd *cobra.Command, args []string) error {
	rc := GetRunContext(cmd)
	follow, _ := cmd.Flags().GetBool("follow")
	combination, _ := cmd.Flags().GetString("combination")
	combination = strings.TrimSpace(combination)

	targets, err := listEvalsetRunTargets(cmd, rc, args[0], combination)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return &cliError{Code: "not_found", Message: "no runs found for this eval set (or combination filter)"}
	}
	if !rc.Output.IsStructured() {
		fmt.Fprintf(os.Stderr, "%s Multiplexing events for %d run(s)\n", output.Cyan("▸"), len(targets))
	}

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	lines := make(chan evalsetLogLine, 64)
	var wg sync.WaitGroup
	for _, t := range targets {
		wg.Add(1)
		go func(t evalsetRunTarget) {
			defer wg.Done()
			streamEvalsetRunEvents(ctx, cmd, rc, t, follow, lines)
		}(t)
	}
	go func() {
		wg.Wait()
		close(lines)
	}()

	failedStreams := 0
	for line := range lines {
		if line.Err != nil {
			failedStreams++
			if !rc.Output.IsStructured() {
				fmt.Fprintf(os.Stderr, "%s [%s] stream error: %v\n", output.Faint("▸"), line.MatrixKey, line.Err)
			}
			// Survive one (or more) run stream failures; keep multiplexing.
			continue
		}
		if line.Done {
			continue
		}
		prefix := line.MatrixKey
		if prefix == "" {
			prefix = line.RunID
		}
		switch {
		case rc.Output.IsStructured():
			envelope := map[string]any{
				"matrix_key": prefix,
				"run_id":     line.RunID,
				"event":      line.Event,
				"id":         line.ID,
			}
			var data any
			if json.Unmarshal(line.Data, &data) == nil {
				envelope["data"] = data
			} else {
				envelope["data"] = string(line.Data)
			}
			out, _ := json.Marshal(envelope)
			fmt.Fprintln(rc.Output.Writer(), string(out))
		default:
			ts := time.Now().Format("15:04:05")
			summary := string(line.Data)
			var parsed map[string]any
			if json.Unmarshal(line.Data, &parsed) == nil {
				if et := eventTypeFromPayload(parsed); et != "" {
					summary = et
				}
			}
			fmt.Fprintf(rc.Output.Writer(), "%s [%s] [%s] %s\n",
				output.Faint(ts),
				output.Cyan(output.SanitizeControl(prefix)),
				output.SanitizeControl(line.Event),
				output.SanitizeControl(summary),
			)
		}
	}
	if failedStreams > 0 && failedStreams >= len(targets) {
		return &cliError{Code: "stream_failed", Message: "all eval-set run event streams failed"}
	}
	return nil
}

type evalsetRunTarget struct {
	RunID     string
	MatrixKey string
}

func listEvalsetRunTargets(cmd *cobra.Command, rc *RunContext, evalSetID, combinationFilter string) ([]evalsetRunTarget, error) {
	detail, err := getEvalSet(cmd, rc, evalSetID)
	if err != nil {
		return nil, err
	}
	sessionIDs := mapSlice(detail, "eval_session_ids")
	targets := make([]evalsetRunTarget, 0)
	for _, raw := range sessionIDs {
		sessionID := fmt.Sprint(raw)
		if sessionID == "" {
			continue
		}
		session, err := getEvalSession(cmd, rc, sessionID)
		if err != nil {
			return nil, fmt.Errorf("load eval session %s: %w", sessionID, err)
		}
		for _, runRaw := range mapSlice(session, "runs") {
			run, _ := runRaw.(map[string]any)
			runID := mapString(run, "id")
			if runID == "" {
				continue
			}
			key := matrixKeyFromRun(run)
			if combinationFilter != "" && key != combinationFilter {
				continue
			}
			targets = append(targets, evalsetRunTarget{RunID: runID, MatrixKey: key})
		}
	}
	return targets, nil
}

func matrixKeyFromRun(run map[string]any) string {
	if key := mapString(run, "matrix_key"); key != "" {
		return key
	}
	plan := mapObject(run, "execution_plan")
	series := mapObject(plan, "series")
	if key := mapString(series, "matrix_key"); key != "" {
		return key
	}
	return mapString(run, "id")
}

func streamEvalsetRunEvents(ctx context.Context, cmd *cobra.Command, rc *RunContext, target evalsetRunTarget, follow bool, out chan<- evalsetLogLine) {
	lastEventID := ""
	for {
		if ctx.Err() != nil {
			return
		}
		ch, err := rc.Client.StreamSSEFrom(ctx, "/v1/runs/"+target.RunID+"/events/stream", nil, lastEventID)
		if err != nil {
			out <- evalsetLogLine{MatrixKey: target.MatrixKey, RunID: target.RunID, Err: err}
			return
		}
		received := false
		for event := range ch {
			received = true
			if event.ID != "" {
				lastEventID = event.ID
			}
			out <- evalsetLogLine{
				MatrixKey: target.MatrixKey,
				RunID:     target.RunID,
				Event:     event.Event,
				ID:        event.ID,
				Data:      event.Data,
			}
		}
		if !follow {
			out <- evalsetLogLine{MatrixKey: target.MatrixKey, RunID: target.RunID, Done: true}
			return
		}
		terminal, probeErr := runReachedTerminalStatus(cmd, rc, target.RunID)
		if probeErr != nil {
			out <- evalsetLogLine{MatrixKey: target.MatrixKey, RunID: target.RunID, Err: probeErr}
			return
		}
		if terminal {
			out <- evalsetLogLine{MatrixKey: target.MatrixKey, RunID: target.RunID, Done: true}
			return
		}
		if !received {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
		}
	}
}
