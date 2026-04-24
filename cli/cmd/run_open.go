package cmd

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/agentclash/agentclash/cli/internal/auth"
	"github.com/spf13/cobra"
)

func init() {
	runCmd.AddCommand(runOpenCmd)
}

// openBrowserFunc is swapped out in tests. Matches auth.OpenBrowser's signature.
var openBrowserFunc = auth.OpenBrowser

var runOpenCmd = &cobra.Command{
	Use:   "open <run-id>",
	Short: "Open the run replay in the web UI",
	Long: `Opens https://agentclash.dev/runs/<id> in your default browser.

In non-interactive environments (no TTY, --json, CI=true, etc.) this
command prints the URL to stdout and exits 0 — handy for piping to other
tools in CI.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rc := GetRunContext(cmd)
		runID := args[0]

		webURL := webRunURL(rc.Config.APIURL(), runID)

		if rc.Output.IsStructured() {
			return rc.Output.PrintRaw(map[string]string{
				"run_id": runID,
				"url":    webURL,
			})
		}

		if !isInteractiveTerminal(rc) || !auth.CanOpenBrowser() {
			fmt.Fprintln(rc.Output.Writer(), webURL)
			return nil
		}

		if err := openBrowserFunc(webURL); err != nil {
			// Fall back to printing the URL rather than failing — the user
			// can still click.
			fmt.Fprintf(rc.Output.Writer(),
				"Could not launch browser (%v). Open manually: %s\n",
				err, webURL,
			)
			return nil
		}
		fmt.Fprintln(rc.Output.Writer(), webURL)
		return nil
	},
}

// webRunURL derives the web URL for a run from the current API base URL so
// that dev installs land on localhost:3000 and prod lands on agentclash.dev.
// Transformation rules (simplest thing that works):
//
//   - https://api.<host> → https://<host>
//   - http://localhost:8080 → http://localhost:3000
//   - any other URL → same host, no mutation (best effort)
func webRunURL(apiURL, runID string) string {
	const defaultWebBase = "https://agentclash.dev"
	parsed, err := url.Parse(apiURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return defaultWebBase + "/runs/" + runID
	}

	host := parsed.Host
	switch {
	case strings.HasPrefix(host, "api."):
		host = strings.TrimPrefix(host, "api.")
	case strings.HasPrefix(host, "staging-api."):
		host = "staging." + strings.TrimPrefix(host, "staging-api.")
	case host == "localhost:8080":
		host = "localhost:3000"
	case strings.HasPrefix(host, "127.0.0.1:8080"):
		host = "127.0.0.1:3000"
	}
	return parsed.Scheme + "://" + host + "/runs/" + runID
}
