package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	Type    string            `json:"type"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// Deregister removes the server from the user-scoped Claude MCP config.
func Deregister(name string, cfg *registry.ServerConfig) error {
	if cfg.Transport == "stdio" && cfg.ManagedBy == "claude" {
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
func Register(name string, cfg *registry.ServerConfig) error {
	if cfg.Transport == "stdio" && cfg.ManagedBy == "claude" {
		fmt.Printf("%s is managed by Claude — no action needed\n", name)
		return nil
	}

	c, err := loadConfig()
	if err != nil {
		return err
	}

	entry := mcpEntry{
		Type: "http",
		URL:  fmt.Sprintf("http://localhost:%d%s", cfg.Port, cfg.EndpointPath),
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

	// Convert struct to map[string]any for JSON merge.
	entryJSON, _ := json.Marshal(entry)
	var entryMap map[string]any
	json.Unmarshal(entryJSON, &entryMap) //nolint:errcheck
	c.mcpServers[name] = entryMap

	if err := c.save(); err != nil {
		return err
	}
	fmt.Printf("registered %s → %s (user config)\n", name, entry.URL)
	return nil
}

// registerArgs builds the `claude mcp add` argument list for an HTTP server.
// Kept for tests — no longer used in production.
func registerArgs(name string, cfg *registry.ServerConfig) []string {
	url := fmt.Sprintf("http://localhost:%d%s", cfg.Port, cfg.EndpointPath)
	args := []string{"mcp", "add", "--transport", "http", "--scope", "user"}
	for _, h := range cfg.Headers {
		args = append(args, "--header", h)
	}
	return append(args, name, url)
}
