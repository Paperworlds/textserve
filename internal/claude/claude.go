package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pdonorio/mcp-fleet/internal/localconfig"
	"github.com/pdonorio/mcp-fleet/internal/op"
	"github.com/pdonorio/mcp-fleet/internal/registry"
)

// configPath returns the absolute path to the user-scoped Claude config.
func configPath() string {
	home := os.Getenv("HOME")
	return filepath.Join(home, ".claude-work", ".claude.json")
}

// claudeConfig represents the subset of .claude.json we care about.
type claudeConfig struct {
	raw        map[string]any
	mcpServers map[string]any
}

func loadConfig() (*claudeConfig, error) {
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	servers, _ := raw["mcpServers"].(map[string]any)
	if servers == nil {
		servers = make(map[string]any)
		raw["mcpServers"] = servers
	}
	return &claudeConfig{raw: raw, mcpServers: servers}, nil
}

func (c *claudeConfig) save() error {
	path := configPath()
	data, err := json.MarshalIndent(c.raw, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

// mcpEntry builds the JSON entry for a server.
type mcpEntry struct {
	Type          string            `json:"type"`
	URL           string            `json:"url,omitempty"`
	Command       string            `json:"command,omitempty"`
	Args          []string          `json:"args,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	DisabledTools []string          `json:"disabledTools,omitempty"`
}

// Deregister removes the server from the user-scoped Claude MCP config.
func Deregister(name string, cfg *registry.ServerConfig) error {
	if cfg.Runtime == "claude" {
		return nil
	}
	c, err := loadConfig()
	if err != nil {
		return err
	}
	delete(c.mcpServers, name)
	return c.save()
}

// Register adds the server to the user-scoped Claude MCP config.
// Writes directly to ~/.claude-work/.claude.json — no shelling out to `claude` CLI.
// For runtime=claude stdio servers, op secrets are resolved at register time so
// Claude Code can inject them when it starts the subprocess.
func Register(name string, cfg *registry.ServerConfig) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	var entry mcpEntry

	if cfg.Protocol == "stdio" {
		// stdio registration: resolve command via venv if configured.
		cmd := cfg.NativeCmd
		if cfg.NativeVenv != "" {
			venv := os.ExpandEnv(cfg.NativeVenv)
			candidate := filepath.Join(venv, "bin", cfg.NativeCmd)
			if _, err := os.Stat(candidate); err == nil {
				cmd = candidate
			}
		}
		args := make([]string, len(cfg.NativeArgs))
		for i, a := range cfg.NativeArgs {
			args[i] = os.ExpandEnv(a)
		}
		// Resolve env vars: local config overrides take priority, then server.yaml
		// static values. op:// references are resolved via `op read`.
		localCfg, _ := localconfig.Load()
		localEnv := localCfg.EnvFor(name)
		var env map[string]string
		for _, e := range cfg.Env {
			ref := localEnv[e.Name] // local config override (op path or plain value)
			if ref == "" {
				ref = e.Op // fall back to server.yaml op field (discouraged for personal paths)
			}
			var val string
			switch {
			case ref != "" && strings.HasPrefix(ref, "op://"):
				if v, err := op.Read(ref); err == nil {
					val = v
				} else {
					fmt.Printf("warning: %v\n", err)
				}
			case ref != "":
				val = os.ExpandEnv(ref)
			case e.Value != "":
				val = os.ExpandEnv(e.Value)
			}
			if val != "" {
				if env == nil {
					env = make(map[string]string)
				}
				env[e.Name] = val
			}
		}
		entry = mcpEntry{Type: "stdio", Command: cmd, Args: args, Env: env, DisabledTools: cfg.DisabledTools}
		fmt.Printf("registered %s → stdio:%s (user config)\n", name, cmd)
	} else {
		// HTTP registration.
		entry = mcpEntry{
			Type:          "http",
			URL:           fmt.Sprintf("http://localhost:%d%s", cfg.Port, cfg.EndpointPath),
			DisabledTools: cfg.DisabledTools,
		}
		// Parse "Key: Value" header strings into a map.
		if len(cfg.Headers) > 0 {
			entry.Headers = make(map[string]string, len(cfg.Headers))
			for _, h := range cfg.Headers {
				parts := strings.SplitN(h, ": ", 2)
				if len(parts) == 2 {
					entry.Headers[parts[0]] = parts[1]
				}
			}
		}
		fmt.Printf("registered %s → %s (user config)\n", name, entry.URL)
	}

	// Convert struct to map[string]any for JSON merge.
	entryJSON, _ := json.Marshal(entry)
	var entryMap map[string]any
	json.Unmarshal(entryJSON, &entryMap) //nolint:errcheck
	c.mcpServers[name] = entryMap

	return c.save()
}

// IsRegistered returns true if the named server exists in the user-scoped
// Claude MCP config (~/.claude-work/.claude.json → mcpServers).
func IsRegistered(name string) bool {
	c, err := loadConfig()
	if err != nil {
		return false
	}
	_, ok := c.mcpServers[name]
	return ok
}

// registerArgs builds the `claude mcp add` argument list for an HTTP server.
// Kept for tests — no longer used in production.
func registerArgs(name string, cfg *registry.ServerConfig) []string {
	url := fmt.Sprintf("http://localhost:%d%s", cfg.Port, cfg.EndpointPath)
	args := []string{"mcp", "add", "--transport", cfg.Protocol, "--scope", "user"}
	for _, h := range cfg.Headers {
		args = append(args, "--header", h)
	}
	return append(args, name, url)
}
