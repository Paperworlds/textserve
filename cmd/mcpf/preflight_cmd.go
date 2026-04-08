package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pdonorio/mcp-fleet/internal/preflight"
)

func newPreflightCmd() *cobra.Command {
	var (
		tagsFlag string
		jsonOut  bool
	)
	cmd := &cobra.Command{
		Use:   "preflight",
		Short: "Check readiness of MCP servers by tag",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, repoRoot, err := loadFleet()
			if err != nil {
				return err
			}

			var tags []string
			if tagsFlag != "" {
				for _, t := range strings.Split(tagsFlag, ",") {
					t = strings.TrimSpace(t)
					if t != "" {
						tags = append(tags, t)
					}
				}
			}

			report, err := preflight.Run(tags, repoRoot)
			if err != nil {
				return err
			}

			if jsonOut {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				if err := enc.Encode(report); err != nil {
					return err
				}
			} else {
				printPreflightReport(cmd, report)
			}

			if !report.Ready {
				return fmt.Errorf("preflight: %d server(s) blocking: %s",
					len(report.Blocking), strings.Join(report.Blocking, ", "))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&tagsFlag, "tags", "", "comma-separated tags to filter servers")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON report to stdout")
	return cmd
}

func printPreflightReport(cmd *cobra.Command, report *preflight.Report) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Preflight  tags=%v  ready=%v\n\n", report.RequestedTags, report.Ready)
	for name, result := range report.Servers {
		icon := statusIcon(result.Status)
		portStr := ""
		if result.Port > 0 {
			portStr = fmt.Sprintf("  port=%d", result.Port)
		}
		errStr := ""
		if result.Error != "" {
			errStr = fmt.Sprintf("  error=%s", result.Error)
		}
		fmt.Fprintf(out, "  %s %-20s  %s%s%s\n", icon, name, result.Status, portStr, errStr)
	}
	if len(report.Blocking) > 0 {
		fmt.Fprintf(out, "\nBlocking: %s\n", strings.Join(report.Blocking, ", "))
	}
}

func statusIcon(status string) string {
	switch status {
	case "healthy":
		return "✓"
	case "unhealthy":
		return "✗"
	case "stopped":
		return "○"
	default:
		return "?"
	}
}
