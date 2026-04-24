package cmd

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/template"
	"time"

	"github.com/agentclash/agentclash/cli/internal/api"
	"github.com/agentclash/agentclash/cli/internal/config"
	"github.com/spf13/cobra"
)

//go:embed templates/AGENTS.md.tmpl
var agentsMdTemplate string

// AgentsMdFile is the filename `link` writes into the project root alongside
// .agentclash.yaml. Kept as a package-level const so tests can clean up.
const AgentsMdFile = "AGENTS.md"

func init() {
	rootCmd.AddCommand(linkCmd)
	linkCmd.Flags().String("org", "", "Organization ID (overrides default; needed only if you belong to multiple orgs)")
}

var linkCmd = &cobra.Command{
	Use:   "link",
	Short: "Link this directory to your AgentClash workspace",
	Long: `Pull the current workspace and its deployments and write a project
config (.agentclash.yaml) + a playbook for AI agents (AGENTS.md).

link is read-only — it does NOT provision, create, or modify any backend
resources. Set up your workspace, provider keys, and deployments in the web
app first (https://agentclash.dev), then run:

  agentclash link                 # interactive workspace picker when needed
  agentclash link -w <id>         # pick explicitly
  agentclash link --json          # machine-readable output

If you need to provision from the terminal (CI, automation), use
` + "`agentclash init --provision`" + ` instead.`,
	RunE: runLink,
}

func runLink(cmd *cobra.Command, _ []string) error {
	rc := GetRunContext(cmd)
	ctx := cmd.Context()

	wsID, err := resolveWorkspaceForLink(cmd, rc)
	if err != nil {
		return err
	}

	ws, err := fetchWorkspaceDetails(ctx, rc.Client, wsID)
	if err != nil {
		return err
	}

	deployments, err := fetchDeployments(ctx, rc.Client, wsID)
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	projectCfg := config.ProjectConfig{
		WorkspaceID:   wsID,
		WorkspaceName: str(ws["name"]),
		OrgID:         str(ws["organization_id"]),
		Deployments:   deployments,
	}
	if err := config.WriteProjectConfig(cwd, projectCfg); err != nil {
		return fmt.Errorf("writing %s: %w", config.ProjectConfigFile, err)
	}

	agentsMdPath := filepath.Join(cwd, AgentsMdFile)
	if err := writeAgentsMd(agentsMdPath, projectCfg); err != nil {
		return fmt.Errorf("writing %s: %w", AgentsMdFile, err)
	}

	if rc.Output.IsStructured() {
		return rc.Output.PrintRaw(map[string]any{
			"workspace_id":   projectCfg.WorkspaceID,
			"workspace_name": projectCfg.WorkspaceName,
			"org_id":         projectCfg.OrgID,
			"deployments":    projectCfg.Deployments,
			"config_path":    config.ProjectConfigFile,
			"agents_md_path": AgentsMdFile,
		})
	}

	depCount := len(deployments)
	names := sortedKeys(deployments)
	if depCount == 0 {
		rc.Output.PrintWarning(fmt.Sprintf(
			"workspace %q has 0 deployments — deploy a model at %s, then rerun `agentclash link`.",
			projectCfg.WorkspaceName, deployURL(wsID),
		))
	} else {
		rc.Output.PrintSuccess(fmt.Sprintf(
			"linked workspace %q (%d deployments: %s)",
			projectCfg.WorkspaceName, depCount, joinNames(names),
		))
	}
	rc.Output.PrintSuccess(fmt.Sprintf("wrote %s", config.ProjectConfigFile))
	rc.Output.PrintSuccess(fmt.Sprintf("wrote %s", AgentsMdFile))
	if depCount > 0 {
		fmt.Fprintln(rc.Output.Writer(), "Ready. Try: agentclash challenge-pack new <name>")
	}
	return nil
}

// resolveWorkspaceForLink returns the workspace id link should pull from.
// Precedence: -w flag / AGENTCLASH_WORKSPACE env / user default > only-one-
// workspace auto-select > interactive picker. Returns a descriptive error if
// none of those apply (e.g., multi-workspace, non-interactive).
func resolveWorkspaceForLink(cmd *cobra.Command, rc *RunContext) (string, error) {
	if rc.Workspace != "" {
		return rc.Workspace, nil
	}

	orgID, _ := cmd.Flags().GetString("org")
	if orgID == "" {
		orgID = rc.Config.OrgID()
	}

	workspaces, err := listWorkspacesForPicker(cmd.Context(), rc.Client, orgID)
	if err != nil {
		return "", err
	}

	switch len(workspaces) {
	case 0:
		return "", fmt.Errorf("no workspaces found. Sign up and create one at https://agentclash.dev")
	case 1:
		return str(workspaces[0]["id"]), nil
	}

	if !isInteractiveTerminal(rc) {
		return "", fmt.Errorf(
			"you have %d workspaces — pass -w <workspace-id> or run interactively to pick one",
			len(workspaces),
		)
	}

	opts := make([]pickerOption, len(workspaces))
	for i, w := range workspaces {
		opts[i] = pickerOption{
			Label:       str(w["name"]),
			Description: fmt.Sprintf("id=%s slug=%s", str(w["id"]), str(w["slug"])),
			Value:       str(w["id"]),
		}
	}
	picked, err := newInteractivePicker().Select("Which workspace do you want to link?", opts)
	if err != nil {
		return "", err
	}
	return picked.Value, nil
}

// listWorkspacesForPicker returns every workspace the caller can see across
// all their orgs. When orgID is non-empty we narrow to that org (skip the
// org list round-trip).
func listWorkspacesForPicker(ctx context.Context, client *api.Client, orgID string) ([]map[string]any, error) {
	if orgID != "" {
		return listWorkspacesInOrg(ctx, client, orgID)
	}

	orgs, err := listOrgs(ctx, client)
	if err != nil {
		return nil, err
	}

	var all []map[string]any
	for _, org := range orgs {
		id := str(org["id"])
		if id == "" {
			continue
		}
		items, err := listWorkspacesInOrg(ctx, client, id)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
	}
	return all, nil
}

func listOrgs(ctx context.Context, client *api.Client) ([]map[string]any, error) {
	resp, err := client.Get(ctx, "/v1/organizations", nil)
	if err != nil {
		return nil, err
	}
	if apiErr := resp.ParseError(); apiErr != nil {
		return nil, apiErr
	}
	var out struct {
		Items []map[string]any `json:"items"`
	}
	if err := resp.DecodeJSON(&out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

func listWorkspacesInOrg(ctx context.Context, client *api.Client, orgID string) ([]map[string]any, error) {
	resp, err := client.Get(ctx, "/v1/organizations/"+orgID+"/workspaces", nil)
	if err != nil {
		return nil, err
	}
	if apiErr := resp.ParseError(); apiErr != nil {
		return nil, apiErr
	}
	var out struct {
		Items []map[string]any `json:"items"`
	}
	if err := resp.DecodeJSON(&out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

func fetchWorkspaceDetails(ctx context.Context, client *api.Client, wsID string) (map[string]any, error) {
	resp, err := client.Get(ctx, "/v1/workspaces/"+wsID+"/details", nil)
	if err != nil {
		return nil, err
	}
	if apiErr := resp.ParseError(); apiErr != nil {
		return nil, apiErr
	}
	var ws map[string]any
	if err := resp.DecodeJSON(&ws); err != nil {
		return nil, err
	}
	return ws, nil
}

// fetchDeployments returns a (deployment name → id) map. Unnamed deployments
// are skipped — eval --models wouldn't be able to reference them by name
// anyway.
func fetchDeployments(ctx context.Context, client *api.Client, wsID string) (map[string]string, error) {
	resp, err := client.Get(ctx, "/v1/workspaces/"+wsID+"/agent-deployments", nil)
	if err != nil {
		return nil, err
	}
	if apiErr := resp.ParseError(); apiErr != nil {
		return nil, apiErr
	}
	var out struct {
		Items []map[string]any `json:"items"`
	}
	if err := resp.DecodeJSON(&out); err != nil {
		return nil, err
	}

	m := make(map[string]string, len(out.Items))
	for _, item := range out.Items {
		name := str(item["name"])
		id := str(item["id"])
		if name == "" || id == "" {
			continue
		}
		m[name] = id
	}
	return m, nil
}

func writeAgentsMd(path string, cfg config.ProjectConfig) error {
	tmpl, err := template.New("agents-md").Parse(agentsMdTemplate)
	if err != nil {
		return err
	}

	data := struct {
		WorkspaceID     string
		WorkspaceName   string
		DeploymentNames []string
		DeployURL       string
		GeneratedAt     string
	}{
		WorkspaceID:     cfg.WorkspaceID,
		WorkspaceName:   cfg.WorkspaceName,
		DeploymentNames: sortedKeys(cfg.Deployments),
		DeployURL:       deployURL(cfg.WorkspaceID),
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}

func deployURL(wsID string) string {
	return fmt.Sprintf("https://agentclash.dev/workspaces/%s/deployments", wsID)
}

func sortedKeys(m map[string]string) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func joinNames(names []string) string {
	buf := bytes.Buffer{}
	for i, n := range names {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(n)
	}
	return buf.String()
}
