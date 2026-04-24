package cmd

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/agentclash/agentclash/cli/internal/output"
	"github.com/spf13/cobra"
)

//go:embed templates/challenge-pack-starter.yaml.tmpl
var challengePackStarterTemplate string

func init() {
	rootCmd.AddCommand(challengePackCmd)
	challengePackCmd.AddCommand(cpListCmd)
	challengePackCmd.AddCommand(cpPublishCmd)
	challengePackCmd.AddCommand(cpValidateCmd)
	challengePackCmd.AddCommand(cpNewCmd)
	challengePackCmd.AddCommand(cpCheckCmd)

	cpNewCmd.Flags().String("dir", ".", "Directory to write the starter pack into")
	cpNewCmd.Flags().Bool("force", false, "Overwrite <name>.yaml if it already exists")
}

var challengePackCmd = &cobra.Command{
	Use:     "challenge-pack",
	Aliases: []string{"cp"},
	Short:   "Manage challenge packs",
}

var cpListCmd = &cobra.Command{
	Use:   "list",
	Short: "List challenge packs",
	RunE: func(cmd *cobra.Command, args []string) error {
		rc := GetRunContext(cmd)
		wsID := RequireWorkspace(cmd)

		resp, err := rc.Client.Get(cmd.Context(), "/v1/workspaces/"+wsID+"/challenge-packs", nil)
		if err != nil {
			return err
		}
		if apiErr := resp.ParseError(); apiErr != nil {
			return apiErr
		}

		var result struct {
			Items []map[string]any `json:"items"`
		}
		if err := resp.DecodeJSON(&result); err != nil {
			return err
		}

		if rc.Output.IsStructured() {
			return rc.Output.PrintRaw(result)
		}

		cols := []output.Column{{Header: "ID"}, {Header: "Name"}, {Header: "Slug"}, {Header: "Status"}, {Header: "Versions"}}
		rows := make([][]string, len(result.Items))
		for i, item := range result.Items {
			versionCount := "0"
			if versions, ok := item["versions"].([]any); ok {
				versionCount = fmt.Sprintf("%d", len(versions))
			}
			rows[i] = []string{
				str(item["id"]),
				str(item["name"]),
				str(item["slug"]),
				output.StatusColor(str(item["lifecycle_status"])),
				versionCount,
			}
		}
		rc.Output.PrintTable(cols, rows)
		return nil
	},
}

var cpPublishCmd = &cobra.Command{
	Use:   "publish <file>",
	Short: "Publish a challenge pack YAML bundle",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rc := GetRunContext(cmd)
		wsID := RequireWorkspace(cmd)

		data, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}

		sp := output.NewSpinner("Publishing challenge pack...", flagQuiet)
		resp, err := rc.Client.PostRaw(cmd.Context(), "/v1/workspaces/"+wsID+"/challenge-packs", "application/octet-stream", bytes.NewReader(data))
		if err != nil {
			sp.StopWithError("Publish failed")
			return err
		}
		if apiErr := resp.ParseError(); apiErr != nil {
			sp.StopWithError("Publish failed")
			return apiErr
		}

		var result map[string]any
		if err := resp.DecodeJSON(&result); err != nil {
			return err
		}

		sp.StopWithSuccess("Published")

		if rc.Output.IsStructured() {
			return rc.Output.PrintRaw(result)
		}

		rc.Output.PrintDetail("Pack ID", str(result["challenge_pack_id"]))
		rc.Output.PrintDetail("Version ID", str(result["challenge_pack_version_id"]))
		return nil
	},
}

// cpCheckCmd is the human-friendly alias for `validate`. Same behaviour,
// different verb — "check" reads more naturally in a humans-first flow.
var cpCheckCmd = &cobra.Command{
	Use:   "check <file>",
	Short: "Sanity-check a challenge pack YAML bundle (alias for validate)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return cpValidateCmd.RunE(cmd, args)
	},
}

var cpNewCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Scaffold a new challenge pack YAML starter",
	Long: `Writes <name>.yaml in the current directory (or --dir) with a
commented starter challenge. Edit the file to add your own challenges,
then:

  agentclash challenge-pack check <name>.yaml
  agentclash eval <name>.yaml --models <model-name>

Re-running with the same name errors out by default so you don't overwrite
a pack you're iterating on — pass --force if you really mean it.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rc := GetRunContext(cmd)
		rawName := args[0]

		slug, err := toPackSlug(rawName)
		if err != nil {
			return err
		}

		dir, _ := cmd.Flags().GetString("dir")
		if dir == "" {
			dir = "."
		}
		path := filepath.Join(dir, slug+".yaml")

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			if _, statErr := os.Stat(path); statErr == nil {
				return fmt.Errorf("%s already exists; pass --force to overwrite", path)
			}
		}

		tmpl, err := template.New("cp-starter").Parse(challengePackStarterTemplate)
		if err != nil {
			return err
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, struct {
			Slug        string
			DisplayName string
		}{
			Slug:        slug,
			DisplayName: toDisplayName(rawName),
		}); err != nil {
			return err
		}

		if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}

		if rc.Output.IsStructured() {
			return rc.Output.PrintRaw(map[string]any{
				"path":  path,
				"slug":  slug,
				"force": force,
			})
		}
		rc.Output.PrintSuccess(fmt.Sprintf("wrote %s", path))
		fmt.Fprintf(rc.Output.Writer(),
			"Next: edit %s, then `agentclash challenge-pack check %s`.\n",
			path, path,
		)
		return nil
	},
}

// toPackSlug normalises a user-supplied pack name into a slug suitable for
// both the filename and the server-side pack.slug field. Slugs are
// lowercase, hyphenated, and cannot be empty or purely punctuation.
func toPackSlug(name string) (string, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return "", fmt.Errorf("pack name cannot be empty")
	}
	re := regexp.MustCompile(`[^a-z0-9]+`)
	slug := re.ReplaceAllString(name, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "", fmt.Errorf("pack name %q produces an empty slug — use letters, numbers, or hyphens", name)
	}
	return slug, nil
}

// toDisplayName keeps the user's capitalisation and spacing for the human-
// readable pack.name field; falls back to the slug when input is punctuation-
// only (toPackSlug has already rejected that case, so this is belt-and-braces).
func toDisplayName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "Untitled pack"
	}
	return trimmed
}

var cpValidateCmd = &cobra.Command{
	Use:   "validate <file>",
	Short: "Validate a challenge pack YAML bundle",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rc := GetRunContext(cmd)
		wsID := RequireWorkspace(cmd)

		data, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}

		resp, err := rc.Client.PostRaw(cmd.Context(), "/v1/workspaces/"+wsID+"/challenge-packs/validate", "application/octet-stream", bytes.NewReader(data))
		if err != nil {
			return err
		}
		if apiErr := resp.ParseError(); apiErr != nil {
			return apiErr
		}

		var result map[string]any
		if err := resp.DecodeJSON(&result); err != nil {
			return err
		}

		if rc.Output.IsStructured() {
			return rc.Output.PrintRaw(result)
		}

		if valid, ok := result["valid"].(bool); ok && valid {
			rc.Output.PrintSuccess("Challenge pack is valid")
		} else {
			rc.Output.PrintError("Challenge pack has errors")
			if errors, ok := result["errors"].([]any); ok {
				for _, e := range errors {
					fmt.Fprintf(os.Stderr, "  - %v\n", e)
				}
			}
			return fmt.Errorf("validation failed")
		}
		return nil
	},
}
