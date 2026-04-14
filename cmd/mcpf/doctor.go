package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pdonorio/mcp-fleet/internal/registry"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Run full diagnostic checks on the MCP fleet",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runDoctor(cmd.OutOrStdout()); err != nil {
				os.Exit(1)
			}
			return nil
		},
	}
}

func runDoctor(out io.Writer) error {
	hasFail := false
	pass := func(msg string) { fmt.Fprintf(out, "[PASS] %s\n", msg) }
	warn := func(msg string) { fmt.Fprintf(out, "[WARN] %s\n", msg) }
	fail := func(msg string) { hasFail = true; fmt.Fprintf(out, "[FAIL] %s\n", msg) }
	miss := func(msg string) { fmt.Fprintf(out, "[MISS] %s\n", msg) }

	// 1. Registry parse
	root, err := findRepoRoot()
	if err != nil {
		fail(fmt.Sprintf("registry.yaml not found: %v", err))
		return fmt.Errorf("doctor found failures")
	}
	fleet, err := registry.Load(filepath.Join(root, "registry.yaml"))
	if err != nil {
		fail(fmt.Sprintf("registry.yaml parse error: %v", err))
		return fmt.Errorf("doctor found failures")
	}
	pass(fmt.Sprintf("registry.yaml parses cleanly (%d servers)", len(fleet.Servers)))

	// 2. Server configs
	names := fleet.ListNames()
	badConfigs := []string{}
	configs := make(map[string]*registry.ServerConfig, len(names))
	for _, n := range names {
		sc, scErr := registry.LoadServer(root, n)
		if scErr != nil {
			// Fall back to registry entry so we can still run other checks.
			entry := fleet.Servers[n]
			sc = &registry.ServerConfig{
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
			badConfigs = append(badConfigs, n)
		}
		configs[n] = sc
	}
	if len(badConfigs) > 0 {
		fail(fmt.Sprintf("server.yaml load errors: %s", strings.Join(badConfigs, ", ")))
	} else {
		pass("all server.yaml files load")
	}

	// 3. Port conflicts
	portMap := map[int][]string{}
	for n, sc := range configs {
		if sc.Port > 0 {
			portMap[sc.Port] = append(portMap[sc.Port], n)
		}
	}
	conflicts := []string{}
	for port, servers := range portMap {
		if len(servers) > 1 {
			conflicts = append(conflicts, fmt.Sprintf("port %d: %s", port, strings.Join(servers, ", ")))
		}
	}
	if len(conflicts) > 0 {
		fail(fmt.Sprintf("port conflicts: %s", strings.Join(conflicts, "; ")))
	} else {
		pass("no port conflicts")
	}

	// 4. Image availability (docker servers only — warns, does not fail)
	for _, n := range names {
		sc := configs[n]
		if sc.Image == "" || sc.Runtime == "process" || sc.Runtime == "claude" {
			continue
		}
		if err := exec.Command("docker", "image", "inspect", sc.Image).Run(); err != nil {
			warn(fmt.Sprintf("image %s not found in local registry (pull needed)", sc.Image))
		}
	}

	// 5. Dep checks
	for _, n := range names {
		sc := configs[n]
		for _, dep := range sc.Deps {
			if err := exec.Command("bash", "-c", dep.Cmd).Run(); err != nil {
				fail(fmt.Sprintf("%s dep: %s failed — %s", n, dep.Cmd, dep.Hint))
			} else {
				pass(fmt.Sprintf("%s dep check passed", n))
			}
		}
	}

	// 6. Cache status
	home := os.Getenv("HOME")
	for _, n := range names {
		sc := configs[n]
		checkedAny := false
		allCached := true
		missing := []string{}

		for _, ev := range sc.Env {
			if ev.Cache != "" {
				checkedAny = true
				parts := strings.SplitN(ev.Cache, "/", 2)
				var cachePath string
				if len(parts) == 2 {
					cachePath = filepath.Join(home, ".cache", "mcp-"+parts[0], parts[1])
				} else {
					cachePath = filepath.Join(home, ".cache", ev.Cache)
				}
				if _, err := os.Stat(cachePath); os.IsNotExist(err) {
					allCached = false
					missing = append(missing, ev.Name)
				}
			}
			if ev.CacheFile != "" {
				checkedAny = true
				cachePath := filepath.Join(home, ".cache", ev.CacheFile)
				if _, err := os.Stat(cachePath); os.IsNotExist(err) {
					allCached = false
					missing = append(missing, ev.Name)
				}
			}
		}

		if checkedAny {
			if allCached {
				pass(fmt.Sprintf("%s credentials cached", n))
			} else {
				miss(fmt.Sprintf("%s credentials not cached (%s) (run: mcpf start %s)", n, strings.Join(missing, ", "), n))
			}
		}
	}

	if hasFail {
		return fmt.Errorf("doctor found failures")
	}
	return nil
}
