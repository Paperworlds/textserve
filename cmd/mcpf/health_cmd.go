package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pdonorio/mcp-fleet/internal/health"
)

func newHealthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health [name]",
		Short: "Probe health of MCP servers",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fleet, repoRoot, err := loadFleet()
			if err != nil {
				return err
			}

			var names []string
			if len(args) == 1 {
				n := args[0]
				if _, ok := fleet.Servers[n]; !ok {
					return fmt.Errorf("unknown server %q", n)
				}
				names = []string{n}
			} else {
				names = fleet.ListNames()
			}

			anyUnhealthy := false
			out := cmd.OutOrStdout()
			for _, n := range names {
				entry := fleet.Servers[n]
				cfg := serverConfig(repoRoot, n, entry)
				status, probeErr := health.Probe(n, cfg)
				switch status {
				case "healthy":
					fmt.Fprintf(out, "✓ %-20s healthy\n", n)
				case "unhealthy":
					anyUnhealthy = true
					fmt.Fprintf(out, "✗ %-20s unhealthy: %v\n", n, probeErr)
				default:
					fmt.Fprintf(out, "? %-20s unknown\n", n)
				}
			}

			if anyUnhealthy {
				return fmt.Errorf("one or more servers are unhealthy")
			}
			return nil
		},
	}
}
