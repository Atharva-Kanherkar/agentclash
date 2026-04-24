package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/agentclash/agentclash/cli/internal/api"
	"github.com/agentclash/agentclash/cli/internal/config"
	"github.com/agentclash/agentclash/cli/internal/output"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(evalCmd)
	evalCmd.Flags().StringSlice("models", nil, "Comma-separated deployment names to race (required; must exist in .agentclash.yaml)")
	evalCmd.Flags().String("name", "", "Optional run name")
	evalCmd.Flags().Duration("timeout", 10*time.Minute, "Give up waiting for terminal status after this long")
	evalCmd.Flags().Bool("allow-partial", false, "Exit 0 even if one or more agents fail mid-run")
	_ = evalCmd.MarkFlagRequired("models")
}

// evalPollInterval controls how often we poll /v1/runs/<id> while waiting
// for terminal status. Kept low so CLI feedback is snappy on fast runs;
// high enough to not hammer the API on slow runs. Exported-via-var so tests
// can shrink it for in-memory server runs.
var evalPollInterval = 2 * time.Second

var evalCmd = &cobra.Command{
	Use:   "eval <pack.yaml>",
	Short: "Race deployed models on a challenge pack and print a scorecard",
	Long: `Publishes the challenge pack, creates a run against every deployment
named in --models, waits for the run to finish, and prints a scorecard.

Prerequisites (run once in each project):
  agentclash auth login
  agentclash link

The --models flag accepts deployment NAMES (not UUIDs) as written in
.agentclash.yaml by ` + "`agentclash link`" + `. Unknown names fail fast with
DEPLOYMENT_NOT_FOUND.

Exit codes:
  0  pass            all agents completed.
  1  fail            run failed, at least one agent did not complete, or
                     (with --allow-partial off) any agent errored.
  2  timeout         --timeout elapsed before the run reached terminal status.`,
	Args: cobra.ExactArgs(1),
	RunE: runEval,
}

func runEval(cmd *cobra.Command, args []string) error {
	rc := GetRunContext(cmd)
	wsID := RequireWorkspace(cmd)
	ctx := cmd.Context()

	packPath := args[0]

	requested, _ := cmd.Flags().GetStringSlice("models")
	if len(requested) == 0 {
		return fmt.Errorf("--models is required — pass one or more comma-separated deployment names from .agentclash.yaml")
	}
	requested = normaliseModelNames(requested)

	projectCfg := config.FindProjectConfig()
	if projectCfg == nil || len(projectCfg.Deployments) == 0 {
		return fmt.Errorf(
			".agentclash.yaml not found or has no deployments. Run `agentclash link` first, " +
				"or deploy a model at https://agentclash.dev",
		)
	}

	deploymentIDs, missing := resolveModelDeployments(requested, projectCfg.Deployments)
	if len(missing) > 0 {
		return newDeploymentNotFoundError(missing, projectCfg.WorkspaceID, projectCfg.Deployments)
	}

	packVersionID, err := publishPackForEval(ctx, rc, wsID, packPath)
	if err != nil {
		return err
	}

	runName, _ := cmd.Flags().GetString("name")
	runID, err := createRunForEval(ctx, rc, wsID, packVersionID, deploymentIDs, runName)
	if err != nil {
		return err
	}

	timeout, _ := cmd.Flags().GetDuration("timeout")
	terminal, waitErr := waitForTerminalRun(ctx, rc.Client, runID, timeout)

	scorecard, _ := fetchRunRanking(ctx, rc.Client, runID)

	if rc.Output.IsStructured() {
		out := map[string]any{
			"pack_version_id":     packVersionID,
			"run_id":              runID,
			"agent_deployment_ids": deploymentIDs,
			"models":              requested,
			"run":                 terminal,
			"ranking":             scorecard,
		}
		if err := rc.Output.PrintRaw(out); err != nil {
			return err
		}
	} else {
		renderEvalScorecard(rc, requested, terminal, scorecard)
	}

	return evalExitDecision(waitErr, terminal, cmd)
}

// normaliseModelNames splits any comma-joined entries and strips whitespace,
// so `--models gpt-5,claude-4.7` and `--models gpt-5 --models claude-4.7`
// and `--models " gpt-5, claude-4.7 "` all yield the same slice.
func normaliseModelNames(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, raw := range in {
		for _, p := range strings.Split(raw, ",") {
			name := strings.TrimSpace(p)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, name)
		}
	}
	return out
}

// resolveModelDeployments maps requested names to deployment ids using the
// (name → id) table `agentclash link` wrote. Missing names are reported so
// we can fail fast with a list, not one-at-a-time.
func resolveModelDeployments(requested []string, deployments map[string]string) (ids []string, missing []string) {
	ids = make([]string, 0, len(requested))
	for _, name := range requested {
		if id, ok := deployments[name]; ok && id != "" {
			ids = append(ids, id)
		} else {
			missing = append(missing, name)
		}
	}
	return ids, missing
}

func newDeploymentNotFoundError(missing []string, workspaceID string, available map[string]string) error {
	var availableNames []string
	for k := range available {
		availableNames = append(availableNames, k)
	}
	sort.Strings(availableNames)
	deployLink := ""
	if workspaceID != "" {
		deployLink = fmt.Sprintf("https://agentclash.dev/workspaces/%s/deployments", workspaceID)
	}
	msg := fmt.Sprintf(
		"deployment(s) not found in this workspace: %s",
		strings.Join(missing, ", "),
	)
	if len(availableNames) > 0 {
		msg += fmt.Sprintf(" — available: %s", strings.Join(availableNames, ", "))
	}
	if deployLink != "" {
		msg += fmt.Sprintf(". Deploy more at %s then rerun `agentclash link`.", deployLink)
	}
	return fmt.Errorf("%s", msg)
}

func publishPackForEval(ctx context.Context, rc *RunContext, wsID, packPath string) (string, error) {
	data, err := os.ReadFile(packPath)
	if err != nil {
		return "", fmt.Errorf("reading pack %s: %w", packPath, err)
	}
	sp := output.NewSpinner("Publishing pack...", flagQuiet)
	resp, err := rc.Client.PostRaw(ctx, "/v1/workspaces/"+wsID+"/challenge-packs", "application/octet-stream", bytes.NewReader(data))
	if err != nil {
		sp.StopWithError("Pack publish failed")
		return "", err
	}
	if apiErr := resp.ParseError(); apiErr != nil {
		sp.StopWithError("Pack publish failed")
		return "", apiErr
	}

	var pub map[string]any
	if err := resp.DecodeJSON(&pub); err != nil {
		return "", err
	}
	versionID := str(pub["challenge_pack_version_id"])
	if versionID == "" {
		return "", fmt.Errorf("publish response missing challenge_pack_version_id")
	}
	sp.StopWithSuccess(fmt.Sprintf("Published pack (version %s)", versionID))
	return versionID, nil
}

func createRunForEval(ctx context.Context, rc *RunContext, wsID, packVersionID string, deploymentIDs []string, name string) (string, error) {
	body := map[string]any{
		"workspace_id":              wsID,
		"challenge_pack_version_id": packVersionID,
		"agent_deployment_ids":      deploymentIDs,
	}
	if name != "" {
		body["name"] = name
	}
	sp := output.NewSpinner(fmt.Sprintf("Creating run (%d agents)...", len(deploymentIDs)), flagQuiet)
	resp, err := rc.Client.Post(ctx, "/v1/runs", body)
	if err != nil {
		sp.StopWithError("Run creation failed")
		return "", err
	}
	if apiErr := resp.ParseError(); apiErr != nil {
		sp.StopWithError("Run creation failed")
		return "", apiErr
	}
	var run map[string]any
	if err := resp.DecodeJSON(&run); err != nil {
		return "", err
	}
	runID := str(run["id"])
	if runID == "" {
		return "", fmt.Errorf("run create response missing id")
	}
	sp.StopWithSuccess(fmt.Sprintf("Created run %s", runID))
	return runID, nil
}

// terminalRunStatuses are the run.status values that end the lifecycle.
// Must stay in sync with backend/internal/domain/run.go.
var terminalRunStatuses = map[string]bool{
	"completed":            true,
	"failed":               true,
	"cancelled":            true,
	"canceled":             true,
	"timed_out":            true,
	"partially_completed":  true,
}

// waitForTerminalRun polls /v1/runs/<id> until status is terminal or timeout.
// Returns the final run payload plus a wait error (context cancellation or
// timeout); the payload is populated on timeout too — callers may still want
// to show the last-seen status.
func waitForTerminalRun(ctx context.Context, client *api.Client, runID string, timeout time.Duration) (map[string]any, error) {
	deadline := time.Now().Add(timeout)
	var last map[string]any
	for {
		resp, err := client.Get(ctx, "/v1/runs/"+runID, nil)
		if err != nil {
			return last, err
		}
		if apiErr := resp.ParseError(); apiErr != nil {
			return last, apiErr
		}
		var run map[string]any
		if decErr := resp.DecodeJSON(&run); decErr != nil {
			return last, decErr
		}
		last = run

		if terminalRunStatuses[strings.ToLower(str(run["status"]))] {
			return run, nil
		}
		if time.Now().After(deadline) {
			return run, fmt.Errorf("timed out after %s waiting for run %s to reach terminal status (last: %s)", timeout, runID, str(run["status"]))
		}
		select {
		case <-ctx.Done():
			return run, ctx.Err()
		case <-time.After(evalPollInterval):
		}
	}
}

func fetchRunRanking(ctx context.Context, client *api.Client, runID string) (map[string]any, error) {
	resp, err := client.Get(ctx, "/v1/runs/"+runID+"/ranking", nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("ranking unavailable: HTTP %d", resp.StatusCode)
	}
	var body map[string]any
	if err := resp.DecodeJSON(&body); err != nil {
		return nil, err
	}
	return body, nil
}

// renderEvalScorecard prints a one-screen humans-readable scorecard: status
// header + per-agent table ordered by the server's ranking.
func renderEvalScorecard(rc *RunContext, modelNames []string, run, rankingPayload map[string]any) {
	w := rc.Output.Writer()
	fmt.Fprintln(w)
	fmt.Fprintln(w, output.Bold("Scorecard"))
	rc.Output.PrintDetail("Run", str(run["id"]))
	rc.Output.PrintDetail("Status", output.StatusColor(str(run["status"])))
	rc.Output.PrintDetail("Models", strings.Join(modelNames, ", "))

	if rankingPayload == nil {
		fmt.Fprintln(w, output.Faint("Ranking not available. Try `agentclash run ranking <id>` later."))
		return
	}
	ranking, _ := rankingPayload["ranking"].(map[string]any)
	items := mapSlice(ranking, "items")
	if len(items) == 0 {
		fmt.Fprintln(w, output.Faint("No ranked agents yet."))
		return
	}

	cols := []output.Column{
		{Header: "Rank"}, {Header: "Agent"}, {Header: "Status"},
		{Header: "Composite"}, {Header: "Correctness"},
		{Header: "Reliability"}, {Header: "Latency"}, {Header: "Cost"},
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

// evalExitDecision maps the result of the wait into a process exit code.
// Gated by --allow-partial for the `partially_completed` case so CI can
// choose whether mid-run flakes are acceptable.
func evalExitDecision(waitErr error, run map[string]any, cmd *cobra.Command) error {
	if waitErr != nil {
		// Timeout or ctx cancel — distinct exit code so CI can retry
		// specifically on flaky infra.
		return &ExitCodeError{Code: 2, Message: waitErr.Error()}
	}
	status := strings.ToLower(str(run["status"]))
	if status == "completed" {
		return nil
	}

	allowPartial, _ := cmd.Flags().GetBool("allow-partial")
	if allowPartial && status == "partially_completed" {
		return nil
	}

	return &ExitCodeError{
		Code:    1,
		Message: fmt.Sprintf("run %s ended in status %s", str(run["id"]), status),
	}
}

