package cmd

import (
	"fmt"
	"time"

	"github.com/agentclash/agentclash/cli/internal/output"
	"github.com/spf13/cobra"
)

func init() {
	runCmd.AddCommand(runShowCmd)
}

var runShowCmd = &cobra.Command{
	Use:   "show <run-id>",
	Short: "Human-readable summary of a run",
	Long: `Fetches run details plus the ranking (when the run has finished)
and prints a single-screen summary: status, duration, per-agent composite
scores, top regressions. Intended for quick "tell me what happened" after
an eval.

For machine-readable output, use --json (which emits the full ranking
payload), or the underlying commands: ` + "`run get`, `run ranking`, `run scorecard`." + `

Exits 0 regardless of run verdict — this command reports, it doesn't gate.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rc := GetRunContext(cmd)
		runID := args[0]
		ctx := cmd.Context()

		runResp, err := rc.Client.Get(ctx, "/v1/runs/"+runID, nil)
		if err != nil {
			return err
		}
		if apiErr := runResp.ParseError(); apiErr != nil {
			return apiErr
		}
		var run map[string]any
		if err := runResp.DecodeJSON(&run); err != nil {
			return err
		}

		// Ranking is only available for completed runs. 202/409 from the
		// ranking endpoint just means "not yet" — surface that softly
		// rather than bailing.
		var ranking map[string]any
		rankResp, rankErr := rc.Client.Get(ctx, "/v1/runs/"+runID+"/ranking", nil)
		if rankErr == nil {
			switch rankResp.StatusCode {
			case 200:
				_ = rankResp.DecodeJSON(&ranking)
			}
		}

		if rc.Output.IsStructured() {
			payload := map[string]any{"run": run}
			if ranking != nil {
				payload["ranking"] = ranking
			}
			return rc.Output.PrintRaw(payload)
		}

		renderRunShow(rc, run, ranking)
		return nil
	},
}

func renderRunShow(rc *RunContext, run map[string]any, ranking map[string]any) {
	w := rc.Output.Writer()
	fmt.Fprintln(w, output.Bold(fmt.Sprintf("Run %s", str(run["id"]))))
	if name := str(run["name"]); name != "" {
		rc.Output.PrintDetail("Name", name)
	}
	rc.Output.PrintDetail("Status", output.StatusColor(str(run["status"])))
	if duration := runDuration(run); duration != "" {
		rc.Output.PrintDetail("Duration", duration)
	}
	if created := str(run["created_at"]); created != "" {
		rc.Output.PrintDetail("Created", created)
	}

	if ranking == nil {
		fmt.Fprintln(w)
		fmt.Fprintln(w, output.Faint("Ranking not available yet — the run may still be in progress."))
		return
	}

	rankingBlock, _ := ranking["ranking"].(map[string]any)
	items := mapSlice(rankingBlock, "items")
	if len(items) == 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, output.Faint("No agents in ranking."))
		return
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, output.Bold("Per-agent ranking"))
	cols := []output.Column{
		{Header: "Rank"},
		{Header: "Agent"},
		{Header: "Status"},
		{Header: "Composite"},
		{Header: "Correctness"},
		{Header: "Reliability"},
		{Header: "Latency"},
		{Header: "Cost"},
	}
	rows := make([][]string, 0, len(items))
	for _, it := range items {
		item, _ := it.(map[string]any)
		rows = append(rows, []string{
			fmtRank(item["rank"]),
			str(item["label"]),
			output.StatusColor(str(item["status"])),
			fmtScore(item["composite_score"]),
			fmtScore(item["correctness_score"]),
			fmtScore(item["reliability_score"]),
			fmtScore(item["latency_score"]),
			fmtScore(item["cost_score"]),
		})
	}
	rc.Output.PrintTable(cols, rows)
}

// runDuration pulls the most-accurate pair of timestamps available on a run
// payload and returns a human-readable duration. Returns "" when neither
// start nor end is known — better to say nothing than show 0s.
func runDuration(run map[string]any) string {
	started := mapString(run, "started_at")
	ended := mapString(run, "finished_at", "completed_at", "ended_at")
	if started == "" || ended == "" {
		return ""
	}
	startT, err := time.Parse(time.RFC3339, started)
	if err != nil {
		return ""
	}
	endT, err := time.Parse(time.RFC3339, ended)
	if err != nil {
		return ""
	}
	d := endT.Sub(startT).Round(time.Second)
	if d <= 0 {
		return ""
	}
	return d.String()
}

func fmtRank(v any) string {
	if v == nil {
		return "-"
	}
	if f, ok := v.(float64); ok {
		return fmt.Sprintf("%d", int(f))
	}
	return str(v)
}
