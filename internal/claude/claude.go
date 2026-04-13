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
	for _, scope := range []string{"local", "project", "user"} {
		exec.Command("claude", "mcp", "remove", name, "-s", scope).Run() //nolint:errcheck
	}
	return nil
}

// Register adds the server to Claude's user-scoped MCP config via `claude mcp add`.
// Uses --scope user so the registration is visible across all projects.
// For stdio servers managed by Claude this is a no-op.
// For native servers with no port, registers as stdio with the native_cmd + native_args.
func Register(name string, cfg *registry.ServerConfig) error {
	if cfg.Transport == "stdio" && cfg.ManagedBy == "claude" {
		fmt.Printf("%s is managed by Claude — no action needed\n", name)
		return nil
	}
	// Native server with no port: register as stdio so Claude can spawn it directly.
	// Use -- to prevent native_args flags (e.g. -m) from being parsed as claude options.
	if cfg.Transport == "native" && cfg.Port == 0 && cfg.NativeCmd != "" {
		cmdArgs := []string{"mcp", "add", "--transport", "stdio", "--scope", "user", name, "--", cfg.NativeCmd}
		cmdArgs = append(cmdArgs, cfg.NativeArgs...)
		c := exec.Command("claude", cmdArgs...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	}
	c := exec.Command("claude", registerArgs(name, cfg)...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// registerArgs builds the `claude mcp add` argument list for an HTTP server.
// Exported for testing.
func registerArgs(name string, cfg *registry.ServerConfig) []string {
	url := fmt.Sprintf("http://localhost:%d%s", cfg.Port, cfg.EndpointPath)
	args := []string{"mcp", "add", "--transport", "http", "--scope", "user"}
	for _, h := range cfg.Headers {
		args = append(args, "--header", h)
	}
	return append(args, name, url)
}
