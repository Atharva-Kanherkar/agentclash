package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/agentclash/agentclash/cli/internal/api"
	"github.com/agentclash/agentclash/cli/internal/config"
	"github.com/agentclash/agentclash/cli/internal/output"
	"github.com/spf13/cobra"
)

// defaultBaselineName is the auto-save / auto-compare bookmark. Picked to
// feel familiar (matches git's default branch name) so a user running
// `agentclash eval` twice in a row sees a regression verdict without
// knowing the baseline system exists.
const defaultBaselineName = "main"

func init() {
	rootCmd.AddCommand(evalCmd)
	evalCmd.Flags().StringSlice("models", nil, "Comma-separated deployment names to race (required; must exist in .agentclash.yaml)")
	evalCmd.Flags().String("name", "", "Optional run name")
	evalCmd.Flags().Duration("timeout", 10*time.Minute, "Give up waiting for terminal status after this long")
	evalCmd.Flags().Bool("allow-partial", false, "Exit 0 even if one or more agents fail mid-run")
	evalCmd.Flags().String("save-as", "", "Save this run as a named baseline for future `--compare-to`. Overrides default auto-save.")
	evalCmd.Flags().String("compare-to", defaultBaselineName, "Compare against this named baseline. Default `main` auto-saves on first eval in the workspace.")
	evalCmd.Flags().Bool("no-baseline", false, "Skip both baseline save and compare — useful when iterating on packs without committing a bookmark.")
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

Baseline behaviour:
  By default eval compares against the baseline named ` + "`main`" + ` in this
  workspace. If ` + "`main`" + ` does not exist yet (first eval) the current run
  is auto-saved as ` + "`main`" + `, so the next eval gets a regression verdict
  automatically.

  --save-as <name>     Save this run as a named baseline (overrides the
                       auto-save-to-main first-run behaviour).
  --compare-to <name>  Compare against a different baseline.
  --no-baseline        Skip baseline save AND compare entirely.

Exit codes:
  0  pass                 all agents completed, no regressions vs baseline.
  1  fail / regression    run failed, at least one agent did not complete
                          (--allow-partial off), or an agent regressed from
                          baseline.
  2  timeout              --timeout elapsed before the run reached terminal status.`,
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

	baselinePlan, err := planBaselineAction(cmd)
	if err != nil {
		return err
	}

	// Fetch the baseline before publishing/running so a missing
	// explicit-compare-to fails fast rather than waiting 60+ seconds for a
	// run just to discover we can't compare against anything.
	var baselineRanking map[string]any
	if baselinePlan.compare != "" {
		baselineRanking, err = fetchBaseline(ctx, rc.Client, wsID, baselinePlan.compare)
		if err != nil {
			if errors.Is(err, errBaselineNotFound) {
				if !baselinePlan.compareIsDefault {
					// User explicitly asked for a baseline that doesn't
					// exist — don't silently swallow.
					return &ExitCodeError{
						Code:    1,
						Message: fmt.Sprintf("baseline %q not found in workspace", baselinePlan.compare),
					}
				}
				// Default `main` is missing: first-run path. Skip compare;
				// we'll auto-save below if baselinePlan permits.
				baselineRanking = nil
			} else {
				return err
			}
		}
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

	regressions := computeRegressions(baselineRanking, scorecard)

	savedBaseline := ""
	if baselinePlan.shouldSave(baselineRanking) && waitErr == nil && isRunCompleted(terminal) {
		target := baselinePlan.saveTarget()
		if err := saveBaseline(ctx, rc.Client, wsID, target, packVersionID, runID, scorecard); err != nil {
			// Save failure is non-fatal — the run succeeded and the user
			// still gets a scorecard. Surface as a warning so CI logs
			// don't silently drop it.
			rc.Output.PrintWarning(fmt.Sprintf("baseline save failed (%v); scorecard still printed", err))
		} else {
			savedBaseline = target
		}
	}

	if rc.Output.IsStructured() {
		out := map[string]any{
			"pack_version_id":      packVersionID,
			"run_id":               runID,
			"agent_deployment_ids": deploymentIDs,
			"models":               requested,
			"run":                  terminal,
			"ranking":              scorecard,
			"baseline": map[string]any{
				"compared_to":     baselinePlan.compare,
				"compared":        baselineRanking != nil,
				"saved_as":        savedBaseline,
				"regressions":     regressions,
				"regression_count": len(regressions),
			},
		}
		if err := rc.Output.PrintRaw(out); err != nil {
			return err
		}
	} else {
		renderEvalScorecard(rc, requested, terminal, scorecard)
		renderBaselineSection(rc, baselinePlan, baselineRanking != nil, savedBaseline, regressions)
	}

	return evalExitDecision(waitErr, terminal, regressions, cmd)
}

// baselinePlan captures the baseline-related flag intent resolved once up
// front so each branch (fetch, save, exit) consults the same plan rather
// than re-reading flags and duplicating precedence logic.
type baselinePlan struct {
	noBaseline       bool
	compare          string // baseline name to fetch, or "" to skip
	compareIsDefault bool   // true when --compare-to was not explicitly set
	saveAs           string // explicit --save-as, or "" to honour auto-save
}

// shouldSave returns true when we should POST a baseline after the run.
// Rules: --no-baseline always wins; an explicit --save-as always saves;
// otherwise auto-save only fires when the compare baseline didn't exist
// (first-run path).
func (p baselinePlan) shouldSave(existingBaseline map[string]any) bool {
	if p.noBaseline {
		return false
	}
	if p.saveAs != "" {
		return true
	}
	return p.compare != "" && existingBaseline == nil
}

// saveTarget returns the baseline name we'll save under — explicit
// --save-as wins, else the compare-to value we'd have auto-saved against.
func (p baselinePlan) saveTarget() string {
	if p.saveAs != "" {
		return p.saveAs
	}
	return p.compare
}

func planBaselineAction(cmd *cobra.Command) (baselinePlan, error) {
	noBaseline, _ := cmd.Flags().GetBool("no-baseline")
	saveAs, _ := cmd.Flags().GetString("save-as")
	compareTo, _ := cmd.Flags().GetString("compare-to")

	if noBaseline && saveAs != "" {
		return baselinePlan{}, fmt.Errorf("--save-as and --no-baseline are mutually exclusive")
	}

	plan := baselinePlan{
		noBaseline:       noBaseline,
		saveAs:           strings.TrimSpace(saveAs),
		compareIsDefault: !cmd.Flags().Changed("compare-to"),
	}
	if !noBaseline {
		plan.compare = strings.TrimSpace(compareTo)
	}
	return plan, nil
}

// errBaselineNotFound is the sentinel fetchBaseline returns when the
// server responds with 404, so callers can branch on "missing" without
// regex'ing error messages.
var errBaselineNotFound = errors.New("baseline not found")

// fetchBaseline returns the baseline's stored ranking (the
// scorecard_snapshot jsonb) or errBaselineNotFound on 404.
func fetchBaseline(ctx context.Context, client *api.Client, wsID, name string) (map[string]any, error) {
	resp, err := client.Get(ctx, "/v1/workspaces/"+wsID+"/baselines/"+name, nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, errBaselineNotFound
	}
	if apiErr := resp.ParseError(); apiErr != nil {
		return nil, apiErr
	}
	var body struct {
		ScorecardSnapshot json.RawMessage `json:"scorecard_snapshot"`
	}
	if err := resp.DecodeJSON(&body); err != nil {
		return nil, err
	}
	if len(body.ScorecardSnapshot) == 0 {
		return nil, nil
	}
	var snap map[string]any
	if err := json.Unmarshal(body.ScorecardSnapshot, &snap); err != nil {
		return nil, fmt.Errorf("parsing baseline scorecard_snapshot: %w", err)
	}
	return snap, nil
}

func saveBaseline(ctx context.Context, client *api.Client, wsID, name, packVersionID, runID string, scorecard map[string]any) error {
	body := map[string]any{
		"pack_version_id":    packVersionID,
		"run_id":             runID,
		"scorecard_snapshot": scorecard,
	}
	resp, err := client.Post(ctx, "/v1/workspaces/"+wsID+"/baselines/"+name, body)
	if err != nil {
		return err
	}
	if apiErr := resp.ParseError(); apiErr != nil {
		return apiErr
	}
	return nil
}

// Regression describes a single (agent-label level) regression. v1 scope
// is binary status drop (baseline=completed → new=not-completed). Score
// thresholds and per-challenge regressions are v2 once the backend
// exposes a stable per-challenge scorecard endpoint.
type Regression struct {
	Agent          string  `json:"agent"`
	BaselineStatus string  `json:"baseline_status"`
	NewStatus      string  `json:"new_status"`
	Reason         string  `json:"reason"`
	BaselineScore  float64 `json:"baseline_composite,omitempty"`
	NewScore       float64 `json:"new_composite,omitempty"`
}

// computeRegressions runs the baseline vs. current-run diff.
//
// v1 rule: an agent label that completed successfully in the baseline but
// reached a non-completed terminal status in the new run is a regression.
// An agent present in baseline but absent entirely from the new run is
// also a regression (it was never attempted this time, silently).
//
// Missing baseline (first eval, --no-baseline, or compare snapshot we
// couldn't parse) returns nil — no comparison, no regressions.
func computeRegressions(baseline, current map[string]any) []Regression {
	if baseline == nil || current == nil {
		return nil
	}
	baselineItems := rankingItems(baseline)
	currentItems := rankingItems(current)
	if len(baselineItems) == 0 {
		return nil
	}
	byLabel := make(map[string]map[string]any, len(currentItems))
	for _, item := range currentItems {
		if label := str(item["label"]); label != "" {
			byLabel[label] = item
		}
	}

	var regs []Regression
	for _, base := range baselineItems {
		label := str(base["label"])
		if label == "" {
			continue
		}
		baseStatus := strings.ToLower(str(base["status"]))
		if baseStatus != "completed" {
			// Baseline itself wasn't a clean pass; new run can't regress
			// against a partial baseline. Users who want strict parity
			// should capture a known-good baseline first.
			continue
		}
		newItem, present := byLabel[label]
		if !present {
			regs = append(regs, Regression{
				Agent:          label,
				BaselineStatus: "completed",
				NewStatus:      "absent",
				Reason:         "agent present in baseline but not in current run",
				BaselineScore:  floatVal(base["composite_score"]),
			})
			continue
		}
		newStatus := strings.ToLower(str(newItem["status"]))
		if newStatus != "completed" {
			regs = append(regs, Regression{
				Agent:          label,
				BaselineStatus: "completed",
				NewStatus:      newStatus,
				Reason:         "status regressed from completed",
				BaselineScore:  floatVal(base["composite_score"]),
				NewScore:       floatVal(newItem["composite_score"]),
			})
		}
	}
	return regs
}

func rankingItems(payload map[string]any) []map[string]any {
	ranking, _ := payload["ranking"].(map[string]any)
	if ranking == nil {
		return nil
	}
	items := mapSlice(ranking, "items")
	if len(items) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		if m, ok := it.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func floatVal(v any) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}

func isRunCompleted(run map[string]any) bool {
	return strings.ToLower(str(run["status"])) == "completed"
}

func renderBaselineSection(rc *RunContext, plan baselinePlan, compared bool, savedAs string, regressions []Regression) {
	w := rc.Output.Writer()
	if plan.noBaseline {
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, output.Bold("Baseline"))
	if compared {
		rc.Output.PrintDetail("Compared to", plan.compare)
	} else if plan.compare != "" {
		rc.Output.PrintDetail("Compared to", plan.compare+" (not yet created)")
	}
	if savedAs != "" {
		rc.Output.PrintDetail("Saved as", savedAs)
	}
	if len(regressions) == 0 {
		if compared {
			rc.Output.PrintSuccess("no regressions")
		}
		return
	}
	rc.Output.PrintError(fmt.Sprintf("%d regression(s) vs baseline", len(regressions)))
	for _, r := range regressions {
		fmt.Fprintf(w, "  - %s: %s -> %s (%s)\n", r.Agent, r.BaselineStatus, r.NewStatus, r.Reason)
	}
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
// choose whether mid-run flakes are acceptable. Regressions (per the
// baseline diff) always fail unless --no-baseline.
func evalExitDecision(waitErr error, run map[string]any, regressions []Regression, cmd *cobra.Command) error {
	if waitErr != nil {
		// Timeout or ctx cancel — distinct exit code so CI can retry
		// specifically on flaky infra.
		return &ExitCodeError{Code: 2, Message: waitErr.Error()}
	}
	status := strings.ToLower(str(run["status"]))

	runOK := status == "completed"
	if !runOK {
		allowPartial, _ := cmd.Flags().GetBool("allow-partial")
		if !(allowPartial && status == "partially_completed") {
			return &ExitCodeError{
				Code:    1,
				Message: fmt.Sprintf("run %s ended in status %s", str(run["id"]), status),
			}
		}
	}

	if len(regressions) > 0 {
		labels := make([]string, 0, len(regressions))
		for _, r := range regressions {
			labels = append(labels, r.Agent)
		}
		return &ExitCodeError{
			Code:    1,
			Message: fmt.Sprintf("REGRESSION_DETECTED vs baseline — %d agent(s) regressed: %s", len(regressions), strings.Join(labels, ", ")),
		}
	}
	return nil
}

