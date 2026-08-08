package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/agentclash/agentclash/runtime/evalset"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// resolveEvalsetManifest rewrites pack/agent refs to UUIDs for POST /v1/eval-sets.
func resolveEvalsetManifest(cmd *cobra.Command, rc *RunContext, workspaceID string, manifest evalset.Manifest) (map[string]any, error) {
	packs := make([]string, 0, len(manifest.Packs))
	for _, ref := range manifest.Packs {
		id, err := resolveEvalsetPackRef(cmd, rc, workspaceID, ref)
		if err != nil {
			return nil, fmt.Errorf("pack %q: %w", ref, err)
		}
		packs = append(packs, id)
	}
	agents := make([]map[string]string, 0, len(manifest.Agents))
	for _, a := range manifest.Agents {
		id, err := resolveEvalsetAgentRef(cmd, rc, workspaceID, a.Deployment)
		if err != nil {
			return nil, fmt.Errorf("agent deployment %q: %w", a.Deployment, err)
		}
		// Omit label: runtime/evalset.agentRef prefers label over deployment, and
		// the API requires agent_ref to be a deployment UUID at create time.
		agents = append(agents, map[string]string{"deployment": id})
	}

	out := map[string]any{
		"schema":      manifest.Schema,
		"name":        manifest.Name,
		"packs":       packs,
		"agents":      agents,
		"models":      manifest.Models,
		"repeats":     manifest.Repeats,
		"case_fanout": manifest.CaseFanout,
		"limits": map[string]any{
			"max_concurrent_runs": manifest.Limits.MaxConcurrentRuns,
			"budget_usd":          manifest.Limits.BudgetUSD,
		},
	}
	if manifest.Seeds != nil {
		out["seeds"] = manifest.Seeds
	}
	if manifest.Repeats == 0 {
		out["repeats"] = evalset.DefaultRepeats
	}
	return out, nil
}

func resolveEvalsetPackRef(cmd *cobra.Command, rc *RunContext, workspaceID, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("empty pack ref")
	}
	if _, err := uuid.Parse(ref); err == nil {
		return ref, nil
	}
	// Local pack file: must already be published — surface a clear error.
	if strings.HasSuffix(strings.ToLower(ref), ".yaml") || strings.HasSuffix(strings.ToLower(ref), ".yml") {
		if _, err := os.Stat(ref); err == nil {
			return "", fmt.Errorf("local pack file %q: publish with `agentclash challenge-pack publish` and use the returned challenge_pack_version_id", filepath.Base(ref))
		}
	}
	if strings.HasPrefix(ref, "catalog/") {
		slug := strings.TrimPrefix(ref, "catalog/")
		return instantiateCatalogPack(cmd, rc, workspaceID, slug)
	}

	packSelector := ref
	versionSelector := ""
	if i := strings.LastIndex(ref, "@"); i > 0 {
		packSelector = ref[:i]
		versionSelector = ref[i+1:]
		// Allow workspace/slug@N by taking the last path segment as pack slug.
		if strings.Contains(packSelector, "/") {
			parts := strings.Split(packSelector, "/")
			packSelector = parts[len(parts)-1]
		}
	}
	resolved, err := resolveChallengePackForEval(cmd, rc, workspaceID, packSelector, versionSelector, "")
	if err != nil {
		// Fall back to catalog instantiate when workspace match fails.
		if id, catErr := instantiateCatalogPack(cmd, rc, workspaceID, packSelector); catErr == nil {
			return id, nil
		}
		return "", err
	}
	return resolved.VersionID, nil
}

func instantiateCatalogPack(cmd *cobra.Command, rc *RunContext, workspaceID, slug string) (string, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return "", fmt.Errorf("empty catalog slug")
	}
	resp, err := rc.Client.Post(cmd.Context(), "/v1/workspaces/"+workspaceID+"/challenge-pack-catalog/"+url.PathEscape(slug)+"/instantiate", nil)
	if err != nil {
		return "", err
	}
	if apiErr := resp.ParseError(); apiErr != nil {
		return "", apiErr
	}
	var result map[string]any
	if err := resp.DecodeJSON(&result); err != nil {
		return "", err
	}
	id := mapString(result, "challenge_pack_version_id")
	if id == "" {
		return "", fmt.Errorf("catalog instantiate for %q returned no challenge_pack_version_id", slug)
	}
	return id, nil
}

func resolveEvalsetAgentRef(cmd *cobra.Command, rc *RunContext, workspaceID, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("empty agent deployment ref")
	}
	if _, err := uuid.Parse(ref); err == nil {
		return ref, nil
	}
	ids, err := resolveDeploymentIDs(cmd, rc, workspaceID, []string{ref})
	if err != nil {
		return "", err
	}
	if len(ids) != 1 {
		return "", fmt.Errorf("expected one deployment for %q", ref)
	}
	return ids[0], nil
}

func getEvalSet(cmd *cobra.Command, rc *RunContext, id string) (map[string]any, error) {
	resp, err := rc.Client.Get(cmd.Context(), "/v1/eval-sets/"+id, nil)
	if err != nil {
		return nil, err
	}
	if apiErr := resp.ParseError(); apiErr != nil {
		return nil, apiErr
	}
	var result map[string]any
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func listEvalSets(cmd *cobra.Command, rc *RunContext, workspaceID string, limit int) (map[string]any, error) {
	query := url.Values{}
	query.Set("workspace_id", workspaceID)
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	resp, err := rc.Client.Get(cmd.Context(), "/v1/eval-sets", query)
	if err != nil {
		return nil, err
	}
	if apiErr := resp.ParseError(); apiErr != nil {
		return nil, apiErr
	}
	var result map[string]any
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func followEvalSet(cmd *cobra.Command, rc *RunContext, evalSetID string, timeout, pollInterval time.Duration) (map[string]any, error) {
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second
	}
	var deadline time.Time
	if timeout > 0 {
		deadline = time.Now().Add(timeout)
	}
	for {
		result, err := getEvalSet(cmd, rc, evalSetID)
		if err != nil {
			return nil, err
		}
		status := mapString(mapObject(result, "eval_set"), "status")
		if evalsetStatusTerminal(status) {
			return result, nil
		}
		if !rc.Output.IsStructured() && !flagQuiet {
			set := mapObject(result, "eval_set")
			pct := evalsetCompletionPercent(result)
			line := fmt.Sprintf("status: %s  combinations: %s  progress: %s",
				status,
				mapString(set, "combination_count"),
				pct,
			)
			if isInteractiveTerminal(rc) {
				fmt.Fprintf(rc.Output.Writer(), "\r%s", padEvalsetStatusLine(line))
			} else {
				fmt.Fprintln(rc.Output.Writer(), line)
			}
		}
		if !deadline.IsZero() && time.Now().Add(pollInterval).After(deadline) {
			return result, &cliError{
				Code:    "follow_timeout",
				Message: fmt.Sprintf("timed out waiting for eval set %s; last status: %s", evalSetID, status),
			}
		}
		select {
		case <-cmd.Context().Done():
			return result, cmd.Context().Err()
		case <-time.After(pollInterval):
		}
	}
}

func padEvalsetStatusLine(line string) string {
	if len(line) < 80 {
		return line + strings.Repeat(" ", 80-len(line))
	}
	return line
}

func evalsetCompletionPercent(result map[string]any) string {
	agg := mapObject(mapObject(result, "result"), "aggregate")
	if agg == nil {
		// aggregate may be raw JSON object nested differently
		if raw, ok := mapObject(result, "result")["aggregate"]; ok {
			switch v := raw.(type) {
			case map[string]any:
				agg = v
			case json.RawMessage:
				_ = json.Unmarshal(v, &agg)
			}
		}
	}
	combos := mapSlice(agg, "combinations")
	if len(combos) == 0 {
		return "-"
	}
	done := 0
	for _, c := range combos {
		m, _ := c.(map[string]any)
		st := strings.ToLower(mapString(m, "status"))
		if st == "completed" || st == "failed" || st == "cancelled" || st == "canceled" {
			done++
		}
	}
	return fmt.Sprintf("%d/%d", done, len(combos))
}
