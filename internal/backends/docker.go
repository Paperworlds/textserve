package backends

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paperworlds/textserve/internal/registry"
)

// DockerBackend talks to a containerized MCP server over HTTP using the
// streamable transport (MCP 2025-03-26). The container is assumed to be
// running already — textserve-mcp does not start it. If the endpoint is not
// listening, ListTools / CallTool surface the error to the caller.
//
// Endpoint is built from the server.yaml: http://localhost:<port><endpoint_path>.
type DockerBackend struct {
	name     string
	endpoint string

	mu      sync.Mutex
	session *mcp.ClientSession
}

// NewDocker builds a DockerBackend from a parsed server.yaml. Returns an
// error if port is missing.
func NewDocker(name string, cfg *registry.ServerConfig) (*DockerBackend, error) {
	if cfg.Port == 0 {
		return nil, fmt.Errorf("docker backend for %q: server.yaml has no port", name)
	}
	path := cfg.EndpointPath
	if path == "" {
		path = "/mcp"
	}
	return &DockerBackend{
		name:     name,
		endpoint: fmt.Sprintf("http://localhost:%d%s", cfg.Port, path),
	}, nil
}

func (b *DockerBackend) Name() string { return b.name }

func (b *DockerBackend) ensureConnected(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.session != nil {
		return nil
	}
	transport := &mcp.StreamableClientTransport{
		Endpoint:             b.endpoint,
		DisableStandaloneSSE: true,
	}
	cli := mcp.NewClient(&mcp.Implementation{
		Name:    "textserve-mcp",
		Version: "dev",
	}, nil)
	session, err := cli.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("connect to %s at %s: %w", b.name, b.endpoint, err)
	}
	b.session = session
	return nil
}

func (b *DockerBackend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.session == nil {
		return nil
	}
	err := b.session.Close()
	b.session = nil
	return err
}

func (b *DockerBackend) ListTools(ctx context.Context) ([]Tool, error) {
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

func (b *DockerBackend) CallTool(ctx context.Context, toolName string, args any) (any, error) {
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
