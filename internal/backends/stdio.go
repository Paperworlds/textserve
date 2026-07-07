package backends

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paperworlds/textserve/internal/registry"
)

// StdioBackend spawns the upstream server as a child process and speaks
// MCP over its stdin/stdout. The child command line is taken from the
// server's server.yaml (native_cmd + native_args).
//
// Construction is cheap; the child is not spawned until first use.
type StdioBackend struct {
	name string
	cmd  string
	args []string

	mu      sync.Mutex
	session *mcp.ClientSession
}

// NewStdio builds an StdioBackend from a parsed server.yaml. Returns an
// error if the config has no native_cmd (which would mean the server isn't
// runnable as a child process).
func NewStdio(name string, cfg *registry.ServerConfig) (*StdioBackend, error) {
	if cfg.NativeCmd == "" {
		return nil, fmt.Errorf("stdio backend for %q: server.yaml has no native_cmd", name)
	}
	return &StdioBackend{
		name: name,
		cmd:  cfg.NativeCmd,
		args: cfg.NativeArgs,
	}, nil
}

func (b *StdioBackend) Name() string { return b.name }

// ensureConnected starts the child process and initialises the MCP session
// the first time it's called. Subsequent calls are no-ops.
func (b *StdioBackend) ensureConnected(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.session != nil {
		return nil
	}
	cmd := exec.Command(b.cmd, b.args...)
	transport := &mcp.CommandTransport{Command: cmd}
	cli := mcp.NewClient(&mcp.Implementation{
		Name:    "textserve-mcp",
		Version: "dev",
	}, nil)
	session, err := cli.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("connect to %s child: %w", b.name, err)
	}
	b.session = session
	return nil
}

func (b *StdioBackend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.session == nil {
		return nil
	}
	err := b.session.Close()
	b.session = nil
	return err
}

func (b *StdioBackend) ListTools(ctx context.Context) ([]Tool, error) {
	if err := b.ensureConnected(ctx); err != nil {
		return nil, err
	}
	res, err := b.session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return nil, fmt.Errorf("%s tools/list: %w", b.name, err)
	}
	out := make([]Tool, 0, len(res.Tools))
	for _, t := range res.Tools {
		out = append(out, Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: schemaToMap(t.InputSchema),
		})
	}
	return out, nil
}

func (b *StdioBackend) CallTool(ctx context.Context, toolName string, args any) (any, error) {
	if err := b.ensureConnected(ctx); err != nil {
		return nil, err
	}
	res, err := b.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	})
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", b.name, toolName, err)
	}
	if res.IsError {
		var msg string
		for _, ct := range res.Content {
			if tc, ok := ct.(*mcp.TextContent); ok {
				msg += tc.Text
			}
		}
		if msg == "" {
			msg = "tool returned isError with no message"
		}
		return nil, fmt.Errorf("%s %s: %s", b.name, toolName, msg)
	}
	if res.StructuredContent != nil {
		raw, err := json.Marshal(res.StructuredContent)
		if err != nil {
			return nil, fmt.Errorf("marshal structured content: %w", err)
		}
		var out any
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, fmt.Errorf("unmarshal structured content: %w", err)
		}
		return out, nil
	}
	var text string
	for _, ct := range res.Content {
		if tc, ok := ct.(*mcp.TextContent); ok {
			text += tc.Text
		}
	}
	return text, nil
}

// schemaToMap normalises whatever InputSchema type the go-sdk exposes
// into a plain map[string]any. Empty/missing schemas return nil.
func schemaToMap(s any) map[string]any {
	if s == nil {
		return nil
	}
	raw, err := json.Marshal(s)
	if err != nil {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}
