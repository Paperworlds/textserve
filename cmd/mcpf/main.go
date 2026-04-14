package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pdonorio/mcp-fleet/internal/claude"
	"github.com/pdonorio/mcp-fleet/internal/deps"
	"github.com/pdonorio/mcp-fleet/internal/docker"
	"github.com/pdonorio/mcp-fleet/internal/native"
	"github.com/pdonorio/mcp-fleet/internal/registry"
)

// version is set at build time via -ldflags, falls back to VERSION file.
var version = ""

func resolveVersion() string {
	if version != "" {
		return version
	}
	root, err := findRepoRoot()
	if err != nil {
		return "unknown"
	}
	data, err := os.ReadFile(filepath.Join(root, "VERSION"))
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(data))
}

func main() {
	root := buildRoot()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func buildRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "mcpf",
		Short: "mcp-fleet management CLI",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	binPath, _ := os.Executable()
	root.Version = fmt.Sprintf("%s (bin: %s)", resolveVersion(), binPath)
	root.AddCommand(
		newStartCmd(),
		newStopCmd(),
		newRestartCmd(),
		newDeregisterCmd(),
		newLogsCmd(),
		newListCmd(),
		newStatusCmd(),
		newHealthCmd(),
		newDoctorCmd(),
		newPreflightCmd(),
		newAddCmd(),
	)
	return root
}

// configFilePath returns the path to the mcpf config file.
func configFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "mcpf", "config.yaml"), nil
}

// readConfigRoot reads the root field from ~/.local/mcpf/config.yaml.
// Returns "" if the file does not exist.
func readConfigRoot() string {
	cfgPath, err := configFilePath()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "root:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "root:"))
			val = strings.Trim(val, `"'`)
			if strings.HasPrefix(val, "~/") {
				home, _ := os.UserHomeDir()
				val = filepath.Join(home, val[2:])
			}
			return val
		}
	}
	return ""
}

// findRepoRoot walks up from cwd until registry.yaml is found,
// then falls back to the root set in ~/.local/mcpf/config.yaml.
func findRepoRoot() (string, error) {
	// Walk up from CWD first.
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "registry.yaml")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	// Config file fallback.
	if root := readConfigRoot(); root != "" {
		if _, err := os.Stat(filepath.Join(root, "registry.yaml")); err == nil {
			return root, nil
		}
	}
	cfgPath, _ := configFilePath()
	return "", fmt.Errorf("registry.yaml not found (set 'root' in %s or run from repo)", cfgPath)
}

func loadFleet() (*registry.FleetRegistry, string, error) {
	root, err := findRepoRoot()
	if err != nil {
		return nil, "", err
	}
	fleet, err := registry.Load(filepath.Join(root, "registry.yaml"))
	if err != nil {
		return nil, "", err
	}
	return fleet, root, nil
}

func resolveNames(fleet *registry.FleetRegistry, name, tag string, all bool) ([]string, error) {
	if all {
		return fleet.AllNames(), nil
	}
	if name != "" {
		if _, ok := fleet.Servers[name]; !ok {
			return nil, fmt.Errorf("unknown server %q", name)
		}
		return []string{name}, nil
	}
	if tag != "" {
		names := fleet.FilterByTag(tag)
		if len(names) == 0 {
			return nil, fmt.Errorf("no servers found with tag %q", tag)
		}
		return names, nil
	}
	return nil, fmt.Errorf("specify a server name, --tag, or --all")
}

// serverConfig loads servers/<name>/server.yaml, falling back to the registry entry.
func serverConfig(repoRoot, name string, entry registry.RegistryEntry) *registry.ServerConfig {
	sc, err := registry.LoadServer(repoRoot, name)
	if err != nil {
		return &registry.ServerConfig{
			Image:         entry.Image,
			Protocol:      entry.Protocol,
			Runtime:       entry.Runtime,
			Port:          entry.Port,
			ContainerPort: entry.ContainerPort,
			EndpointPath:  entry.EndpointPath,
			Tags:          entry.Tags,
			Deps:          entry.Deps,
			Health:        entry.Health,
		}
	}
	return sc
}

// resolvePreStart converts a relative pre_start path to absolute.
func resolvePreStart(repoRoot string, cfg *registry.ServerConfig) {
	if cfg.PreStart != "" && !filepath.IsAbs(cfg.PreStart) {
		cfg.PreStart = filepath.Join(repoRoot, cfg.PreStart)
	}
}

func newStartCmd() *cobra.Command {
	var tag string
	var all bool
	cmd := &cobra.Command{
		Use:   "start [name]",
		Short: "Start one or more MCP servers",
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

				if cfg.Runtime == "claude" {
					fmt.Printf("%s is managed by Claude — no action needed\n", n)
					continue
				}

				startErr := func() error {
					if err := deps.Check(cfg.Deps); err != nil {
						return fmt.Errorf("%s: %w", n, err)
					}
					switch cfg.Runtime {
					case "process":
						// Stop any existing process before starting a new one.
						if status, _ := native.Status(n, cfg); status == "running" {
							_ = native.Stop(n, cfg)
						}
						if err := native.Start(n, cfg); err != nil {
							return fmt.Errorf("start %s: %w", n, err)
						}
					default: // docker
						if err := docker.Run(n, cfg); err != nil {
							return fmt.Errorf("run %s: %w", n, err)
						}
					}
					claude.Deregister(n, cfg) //nolint:errcheck — clear stale entry before re-adding
					if err := claude.Register(n, cfg); err != nil {
						return fmt.Errorf("register %s: %w", n, err)
					}
					return nil
				}()

				if startErr != nil {
					if all {
						fmt.Fprintf(os.Stderr, "skip %s: %v\n", n, startErr)
						failures = append(failures, n)
						continue
					}
					return startErr
				}
				fmt.Printf("started %s\n", n)
			}
			if len(failures) > 0 {
				return fmt.Errorf("failed to start: %s", strings.Join(failures, ", "))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&tag, "tag", "", "start all servers with this tag")
	cmd.Flags().BoolVar(&all, "all", false, "start all servers in the fleet")
	return cmd
}

func newStopCmd() *cobra.Command {
	var tag string
	var all bool
	cmd := &cobra.Command{
		Use:   "stop [name]",
		Short: "Stop one or more MCP servers",
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

				if cfg.Runtime == "claude" {
					fmt.Printf("%s is managed by Claude — no action needed\n", n)
					continue
				}

				if err := claude.Deregister(n, cfg); err != nil {
					fmt.Fprintf(os.Stderr, "deregister %s: %v\n", n, err)
				}

				var stopErr error
				switch cfg.Runtime {
				case "process":
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
	cmd.Flags().StringVar(&tag, "tag", "", "stop all servers with this tag")
	cmd.Flags().BoolVar(&all, "all", false, "stop all servers in the fleet")
	return cmd
}

func newRestartCmd() *cobra.Command {
	var tag string
	var all bool
	cmd := &cobra.Command{
		Use:   "restart [name]",
		Short: "Restart one or more MCP servers",
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
				resolvePreStart(repoRoot, cfg)

				if cfg.Runtime == "claude" {
					fmt.Printf("%s is managed by Claude — no action needed\n", n)
					continue
				}

				// Stop phase (errors non-fatal).
				if err := claude.Deregister(n, cfg); err != nil {
					fmt.Fprintf(os.Stderr, "deregister %s: %v\n", n, err)
				}
				switch cfg.Runtime {
				case "process":
					_ = native.Stop(n, cfg)
				default:
					_ = docker.Stop(n)
				}

				// Check deps before restarting.
				if err := deps.Check(cfg.Deps); err != nil {
					return fmt.Errorf("%s: %w", n, err)
				}

				// Start phase.
				switch cfg.Runtime {
				case "process":
					if err := native.Start(n, cfg); err != nil {
						return fmt.Errorf("restart %s: %w", n, err)
					}
				default:
					if err := docker.Run(n, cfg); err != nil {
						return fmt.Errorf("restart %s: %w", n, err)
					}
				}
				if err := claude.Register(n, cfg); err != nil {
					return fmt.Errorf("register %s: %w", n, err)
				}
				fmt.Printf("restarted %s\n", n)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&tag, "tag", "", "restart all servers with this tag")
	cmd.Flags().BoolVar(&all, "all", false, "restart all servers in the fleet")
	return cmd
}

func newDeregisterCmd() *cobra.Command {
	var tag string
	var all bool
	cmd := &cobra.Command{
		Use:   "deregister [name]",
		Short: "Remove MCP server(s) from Claude without stopping them",
		Long:  "Deregisters one or all MCP servers from Claude's tool list. Does not stop containers. Use --all for a clean-context chat.",
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
				if err := claude.Deregister(n, cfg); err != nil {
					fmt.Fprintf(os.Stderr, "deregister %s: %v\n", n, err)
					continue
				}
				fmt.Printf("deregistered %s\n", n)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&tag, "tag", "", "deregister all servers with this tag")
	cmd.Flags().BoolVar(&all, "all", false, "deregister all servers (clean context)")
	return cmd
}

func newLogsCmd() *cobra.Command {
	var follow bool
	cmd := &cobra.Command{
		Use:   "logs <name>",
		Short: "Show logs for an MCP server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return docker.Logs(args[0], follow)
		},
	}
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "follow log output")
	return cmd
}

func newListCmd() *cobra.Command {
	var tag string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List MCP servers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fleet, _, err := loadFleet()
			if err != nil {
				return err
			}
			var names []string
			if tag != "" {
				names = fleet.FilterByTag(tag)
			} else {
				names = fleet.ListNames()
			}
			out := cmd.OutOrStdout()
			for _, n := range names {
				fmt.Fprintln(out, n)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&tag, "tag", "", "filter by tag")
	return cmd
}
