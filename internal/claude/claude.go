package claude

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/pdonorio/mcp-fleet/internal/registry"
)

// Deregister removes the server from Claude's MCP config via `claude mcp remove`.
// Tries both local and project scopes so it works regardless of where it was registered.
// For stdio servers managed by Claude this is a no-op.
func Deregister(name string, cfg *registry.ServerConfig) error {
	if cfg.Transport == "stdio" && cfg.ManagedBy == "claude" {
		fmt.Printf("%s is managed by Claude — no action needed\n", name)
		return nil
	}
	// Remove from both scopes; ignore errors (server may not exist in a scope).
	for _, scope := range []string{"local", "project"} {
		exec.Command("claude", "mcp", "remove", name, "-s", scope).Run() //nolint:errcheck
	}
	return nil
}

// Register adds the server to Claude's local MCP config via `claude mcp add`.
// Uses --scope local so it never conflicts with project-scoped entries.
// For stdio servers managed by Claude this is a no-op.
func Register(name string, cfg *registry.ServerConfig) error {
	if cfg.Transport == "stdio" && cfg.ManagedBy == "claude" {
		fmt.Printf("%s is managed by Claude — no action needed\n", name)
		return nil
	}
	url := fmt.Sprintf("http://localhost:%d%s", cfg.Port, cfg.EndpointPath)
	c := exec.Command("claude", "mcp", "add", "--transport", "http", "--scope", "local", name, url)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
