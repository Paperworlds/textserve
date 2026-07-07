package backends

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/paperworlds/textserve/internal/registry"
)

// ToolsAPIBackend talks to the local FastAPI tool gateway at
// http://localhost:<port>. Unlike the other backends, this one does not
// speak MCP — the gateway has a custom REST surface:
//
//	GET  /describe         → { tools: { <adapter>: { description, actions: { <action>: { params: {...} } } } } }
//	POST /invoke           → { tool, action, params } → { tool, action, result, cached } or error
//
// Each (adapter, action) pair is exposed as a separate MCP tool, named
// "<adapter>.<action>" (the bundle gating layer prefixes this with the
// registry server name, yielding e.g. "tools-api.snowflake.query").
type ToolsAPIBackend struct {
	name    string
	baseURL string
	http    *http.Client
}

// NewToolsAPI builds a ToolsAPIBackend from a parsed server.yaml. Returns an
// error if port is missing.
func NewToolsAPI(name string, cfg *registry.ServerConfig) (*ToolsAPIBackend, error) {
	if cfg.Port == 0 {
		return nil, fmt.Errorf("tools-api backend for %q: server.yaml has no port", name)
	}
	return &ToolsAPIBackend{
		name:    name,
		baseURL: fmt.Sprintf("http://localhost:%d", cfg.Port),
		http:    &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (b *ToolsAPIBackend) Name() string { return b.name }

func (b *ToolsAPIBackend) Close() error { return nil }

type describeResponse struct {
	Tools map[string]struct {
		Description string                       `json:"description"`
		Actions     map[string]toolsAPIActionDef `json:"actions"`
	} `json:"tools"`
}

type toolsAPIActionDef struct {
	Params map[string]toolsAPIParamDef `json:"params"`
}

type toolsAPIParamDef struct {
	Type     string `json:"type"`
	Required bool   `json:"required"`
	Example  any    `json:"example,omitempty"`
}

func (b *ToolsAPIBackend) ListTools(ctx context.Context) ([]Tool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.baseURL+"/describe", nil)
	if err != nil {
		return nil, err
	}
	resp, err := b.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s GET /describe: %w", b.name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s GET /describe: status %d", b.name, resp.StatusCode)
	}
	var dr describeResponse
	if err := json.NewDecoder(resp.Body).Decode(&dr); err != nil {
		return nil, fmt.Errorf("%s decode /describe: %w", b.name, err)
	}
	var out []Tool
	for adapter, info := range dr.Tools {
		for action, def := range info.Actions {
			out = append(out, Tool{
				Name:        adapter + "." + action,
				Description: info.Description,
				InputSchema: paramsToSchema(def.Params),
			})
		}
	}
	return out, nil
}

func paramsToSchema(params map[string]toolsAPIParamDef) map[string]any {
	props := map[string]any{}
	var required []string
	for pname, pdef := range params {
		typ := pdef.Type
		if typ == "" {
			typ = "string"
		}
		entry := map[string]any{"type": typ}
		if pdef.Example != nil {
			entry["examples"] = []any{pdef.Example}
		}
		props[pname] = entry
		if pdef.Required {
			required = append(required, pname)
		}
	}
	schema := map[string]any{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

type invokeRequest struct {
	Tool   string         `json:"tool"`
	Action string         `json:"action"`
	Params map[string]any `json:"params"`
}

type invokeResponse struct {
	Tool   string `json:"tool"`
	Action string `json:"action"`
	Cached bool   `json:"cached,omitempty"`
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

func (b *ToolsAPIBackend) CallTool(ctx context.Context, toolName string, args any) (any, error) {
	// toolName is "<adapter>.<action>" (the bundle layer's "<server>." prefix
	// has been stripped before reaching here).
	adapter, action, ok := splitFirst(toolName, ".")
	if !ok {
		return nil, fmt.Errorf("%s: tool name %q is not in <adapter>.<action> form", b.name, toolName)
	}
	params, _ := args.(map[string]any)
	if params == nil {
		params = map[string]any{}
	}
	body, err := json.Marshal(invokeRequest{Tool: adapter, Action: action, Params: params})
	if err != nil {
		return nil, fmt.Errorf("marshal invoke body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.baseURL+"/invoke", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")
	resp, err := b.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s POST /invoke: %w", b.name, err)
	}
	defer resp.Body.Close()
	var ir invokeResponse
	if err := json.NewDecoder(resp.Body).Decode(&ir); err != nil {
		return nil, fmt.Errorf("%s decode /invoke response: %w", b.name, err)
	}
	if resp.StatusCode != http.StatusOK {
		if ir.Error != "" {
			return nil, fmt.Errorf("%s %s.%s: %s (status %d)", b.name, adapter, action, ir.Error, resp.StatusCode)
		}
		return nil, fmt.Errorf("%s %s.%s: status %d", b.name, adapter, action, resp.StatusCode)
	}
	return ir.Result, nil
}

func splitFirst(s, sep string) (string, string, bool) {
	for i := 0; i+len(sep) <= len(s); i++ {
		if s[i:i+len(sep)] == sep {
			return s[:i], s[i+len(sep):], true
		}
	}
	return s, "", false
}
