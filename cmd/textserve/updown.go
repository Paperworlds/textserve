package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/paperworlds/textserve/internal/claude"
	"github.com/paperworlds/textserve/internal/docker"
	"github.com/paperworlds/textserve/internal/health"
	"github.com/paperworlds/textserve/internal/native"
	"github.com/paperworlds/textserve/internal/registry"
)

func newUpCmd() *cobra.Command {
	var tag string
	var all bool
	cmd := &cobra.Command{
		Use:   "up [name[,name...]]",
		Short: "Bring one or more MCP servers to desired state (start + register)",
		Long:  "Starts and registers servers that are not running. Skips servers that are already running and registered. Use 'start --force' to restart a running server.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fleet, repoRoot, err := loadFleet()
			if err != nil {
				return err
			}
			name := ""
			if len(args) > 0 {
				name = args[0]
			}
			names, err := resolveNames(fleet, name, tag, all)
			if err != nil {
				return err
			}

			var failures []string
			for _, n := range names {
				entry := fleet.Servers[n]
				cfg := serverConfig(repoRoot, n, entry)
				resolvePreStart(repoRoot, cfg)

				action, upErr := upServer(n, cfg, repoRoot, entry)
				if upErr != nil {
					if all || tag != "" {
						fmt.Fprintf(os.Stderr, "skip %s: %v\n", n, upErr)
						failures = append(failures, n)
						continue
					}
					return upErr
				}
				switch action {
				case "skipped":
					fmt.Printf("%s already running and registered — skipping\n", n)
				case "registered":
					fmt.Printf("registered %s\n", n)
				case "started":
					fmt.Printf("started %s\n", n)
				}
			}
			if len(failures) > 0 {
				return fmt.Errorf("failed: %s", strings.Join(failures, ", "))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&tag, "tag", "", "bring up all servers with this tag")
	cmd.Flags().BoolVar(&all, "all", false, "bring up all servers in the fleet")
	return cmd
}

// upServer converges a single server to the running+registered state.
func upServer(n string, cfg *registry.ServerConfig, repoRoot string, entry registry.RegistryEntry) (string, error) {
	// claude runtime: delegate to startServer which handles shouldRegister logic.
	if cfg.Runtime == registry.RuntimeClaude {
		return startServer(n, cfg, repoRoot, entry, false)
	}

	runStatus := runtimeStatus(n, cfg)
	isReg := claude.IsRegistered(n)

	if runStatus == health.StatusRunning && isReg {
		return "skipped", nil
	}

	if runStatus == health.StatusRunning && !isReg {
		// Running but not registered — just register.
		claude.Deregister(n, cfg) //nolint:errcheck
		if err := claude.Register(n, cfg); err != nil {
			return "", fmt.Errorf("register %s: %w", n, err)
		}
		afterRegister(repoRoot, n, entry)
		return "registered", nil
	}

	// Not running — full start.
	return startServer(n, cfg, repoRoot, entry, false)
}

func newDownCmd() *cobra.Command {
	var tag string
	var all bool
	cmd := &cobra.Command{
		Use:   "down [name[,name...]]",
		Short: "Stop and deregister one or more MCP servers",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fleet, repoRoot, err := loadFleet()
			if err != nil {
				return err
			}
			name := ""
			if len(args) > 0 {
				name = args[0]
			}
			names, err := resolveNames(fleet, name, tag, all)
			if err != nil {
				return err
			}

			for _, n := range names {
				entry := fleet.Servers[n]
				cfg := serverConfig(repoRoot, n, entry)

				if cfg.Runtime == registry.RuntimeClaude {
					fmt.Printf("%s is managed by Claude — no action needed\n", n)
					continue
				}

				if err := claude.Deregister(n, cfg); err != nil {
					fmt.Fprintf(os.Stderr, "deregister %s: %v\n", n, err)
				}
				var stopErr error
				switch cfg.Runtime {
				case registry.RuntimeProcess:
					stopErr = native.Stop(n, cfg)
				default:
					stopErr = docker.Stop(n)
				}
				if stopErr != nil {
					fmt.Fprintf(os.Stderr, "stop %s: %v\n", n, stopErr)
					continue
				}
				fmt.Printf("stopped %s\n", n)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&tag, "tag", "", "bring down all servers with this tag")
	cmd.Flags().BoolVar(&all, "all", false, "bring down all servers in the fleet")
	return cmd
}
