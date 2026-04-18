package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/paperworlds/textserve/internal/op"
	"github.com/paperworlds/textserve/internal/registry"
)

// ResolveEnv processes cfg.Env in order and returns "NAME=VALUE" strings.
func ResolveEnv(cfg *registry.ServerConfig) ([]string, error) {
	resolved := make(map[string]string)
	var result []string

	for _, ev := range cfg.Env {
		var (
			val string
			err error
		)

		switch {
		case ev.Value != "":
			val = ev.Value

		case ev.ValueTemplate != "":
			val = expandVars(ev.ValueTemplate, resolved)

		case ev.Op != "" && ev.Cache != "":
			// cache key format: "service/field"
			parts := strings.SplitN(ev.Cache, "/", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid cache key %q for %s: want service/field", ev.Cache, ev.Name)
			}
			val, err = op.Cached(parts[0], parts[1], ev.Op)
			if err != nil {
				return nil, fmt.Errorf("resolve %s: %w", ev.Name, err)
			}

		case ev.CacheFile != "":
			val, err = op.CacheFileRead(ev.CacheFile)
			if err != nil {
				return nil, fmt.Errorf("resolve %s: %w", ev.Name, err)
			}

		default:
			return nil, fmt.Errorf("env var %q has no resolvable source", ev.Name)
		}

		resolved[ev.Name] = val
		result = append(result, ev.Name+"="+val)
	}

	return result, nil
}

// ResolveVolumes processes cfg.Volumes and returns "host:container[:ro]" strings.
func ResolveVolumes(cfg *registry.ServerConfig) ([]string, error) {
	var result []string
	for _, v := range cfg.Volumes {
		host := os.ExpandEnv(v.Host)
		if v.ResolveSymlinks {
			resolved, err := filepath.EvalSymlinks(host)
			if err != nil {
				return nil, fmt.Errorf("resolve symlinks for %q: %w", host, err)
			}
			host = resolved
		}
		entry := host + ":" + v.Container
		if v.Readonly {
			entry += ":ro"
		}
		result = append(result, entry)
	}
	return result, nil
}

// Run starts the named server as a Docker container.
// If cfg.PreStart is set it is executed (as a bash script) and must succeed.
func Run(name string, cfg *registry.ServerConfig) error {
	if cfg.PreStart != "" {
		c := exec.Command("bash", cfg.PreStart)
		c.Stdout = os.Stderr
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("pre_start %q: %w", cfg.PreStart, err)
		}
	}

	// Remove any existing container (ignore error — may not exist).
	_ = exec.Command("docker", "rm", "-f", "mcp-"+name).Run()

	envVars, err := ResolveEnv(cfg)
	if err != nil {
		return err
	}
	volumes, err := ResolveVolumes(cfg)
	if err != nil {
		return err
	}

	// Build a map of resolved env values for extra_args expansion.
	resolvedMap := make(map[string]string, len(envVars))
	for _, kv := range envVars {
		if idx := strings.IndexByte(kv, '='); idx > 0 {
			resolvedMap[kv[:idx]] = kv[idx+1:]
		}
	}

	args := []string{
		"run", "-d",
		"-p", fmt.Sprintf("%d:%d", cfg.Port, cfg.ContainerPort),
		"--name", "mcp-" + name,
	}
	if cfg.Network != "" {
		args = append(args, "--network", cfg.Network)
	}
	for _, e := range envVars {
		args = append(args, "-e", e)
	}
	for _, v := range volumes {
		args = append(args, "-v", v)
	}
	args = append(args, cfg.Image)
	for _, a := range cfg.ExtraArgs {
		args = append(args, expandVars(a, resolvedMap))
	}

	c := exec.Command("docker", args...)
	c.Stdout = os.Stderr
	c.Stderr = os.Stderr
	return c.Run()
}

// Stop removes the named container.
func Stop(name string) error {
	return exec.Command("docker", "rm", "-f", "mcp-"+name).Run()
}

// Status returns "running", "stopped", or "unknown".
func Status(name string) (string, error) {
	out, err := exec.Command("docker", "inspect", "--format", "{{.State.Status}}", "mcp-"+name).Output()
	if err != nil {
		return "unknown", nil
	}
	if strings.TrimSpace(string(out)) == "running" {
		return "running", nil
	}
	return "stopped", nil
}

// Logs streams the container log to stdout.
func Logs(name string, follow bool) error {
	args := []string{"logs"}
	if follow {
		args = append(args, "--follow")
	}
	args = append(args, "mcp-"+name)
	c := exec.Command("docker", args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// expandVars replaces ${VAR} in s using the provided map.
func expandVars(s string, vars map[string]string) string {
	return os.Expand(s, func(key string) string {
		return vars[key]
	})
}
