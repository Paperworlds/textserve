// Package backends defines the abstraction over the different ways
// textserve-mcp talks to upstream MCP-compatible servers.
//
// Three concrete backends are planned:
//
//	StdioBackend     spawn a child process; speak MCP over stdin/stdout
//	DockerBackend    speak MCP over Streamable HTTP to a running container (3c-iii)
//	ToolsAPIBackend  speak the custom /invoke protocol to tools-api on :10893 (3c-iv)
//
// Each backend lazily connects on first ListTools/CallTool. A backend is
// long-lived: subsequent calls reuse the underlying session.
package backends

import "context"

// Tool is the minimal description of a tool we relay to Claude via tools/list.
type Tool struct {
	Name        string
	Description string
	// InputSchema is the upstream's raw JSON schema for the tool's arguments.
	// We pass this through to Claude verbatim; textserve-mcp does not inspect it.
	InputSchema map[string]any
}

// Backend is the upstream-server abstraction. Implementations must be
// goroutine-safe — Claude may call tools concurrently within a session.
type Backend interface {
	// Name returns the server name (e.g. "textmap", "jenkins"). Used as the
	// tool-name prefix when bundle gating exposes the backend's tools.
	Name() string

	// ListTools returns the upstream's tools/list. Connect-on-first-use.
	ListTools(ctx context.Context) ([]Tool, error)

	// CallTool dispatches a tools/call upstream. The returned value is the
	// upstream's StructuredContent (or concatenated TextContent if the upstream
	// only returned text).
	CallTool(ctx context.Context, toolName string, args any) (any, error)

	// Close releases any connection/child process held by the backend.
	Close() error
}
