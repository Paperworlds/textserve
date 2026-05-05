package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/paperworlds/textserve/internal/claude"
)

func newRemoveCmd() *cobra.Command {
	var (
		global  bool
		repo    string
		all     bool
		dryRun  bool
	)
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove an MCP server entry from a Claude config file",
		Long: `Remove a named MCP server from a Claude JSON config without touching other entries.

Scope flags (default: global):
  (none)         remove from the global config (~/.claude-work/.claude.json or $CLAUDE_CONFIG_DIR)
  --global       same as default — always the global config
  --repo <path>  remove from <path>/.claude/settings.json (project-scoped config)
  --all          remove from all known locations (global + CWD project config if present)

Exits 0 if the entry is not found (idempotent). Use --dry-run to preview without writing.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			var targets []removeTarget

			if all {
				targets = append(targets, removeTarget{
					path:  claude.GlobalConfigPath(),
					label: "global",
				})
				cwd, err := os.Getwd()
				if err == nil {
					p := claude.ProjectConfigPath(cwd)
					if fileExists(p) {
						targets = append(targets, removeTarget{path: p, label: "project (cwd)"})
					}
				}
			} else if repo != "" {
				p := claude.ProjectConfigPath(repo)
				if !fileExists(p) {
					return fmt.Errorf("no .claude/settings.json found at %s", repo)
				}
				targets = []removeTarget{{path: p, label: fmt.Sprintf("project (%s)", repo)}}
			} else {
				// --global or default
				targets = []removeTarget{{path: claude.GlobalConfigPath(), label: "global"}}
			}

			anyFound := false
			for _, t := range targets {
				if dryRun {
					fmt.Printf("[dry-run] would remove %q from %s config: %s\n", name, t.label, t.path)
					continue
				}
				found, err := claude.RemoveFromFile(name, t.path)
				if err != nil {
					return fmt.Errorf("%s: %w", t.label, err)
				}
				if found {
					anyFound = true
					fmt.Printf("removed %q from %s config: %s\n", name, t.label, t.path)
				} else {
					fmt.Printf("%q not found in %s config: %s\n", name, t.label, t.path)
				}
			}
			_ = anyFound
			return nil
		},
	}
	cmd.Flags().BoolVar(&global, "global", false, "remove from the global Claude config (default)")
	cmd.Flags().StringVar(&repo, "repo", "", "remove from a project's .claude/settings.json at this path")
	cmd.Flags().BoolVar(&all, "all", false, "remove from all known config files")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be removed without writing")
	return cmd
}

type removeTarget struct {
	path  string
	label string
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
