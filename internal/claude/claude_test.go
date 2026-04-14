package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pdonorio/mcp-fleet/internal/registry"
)

// setupFakeConfig creates a temporary .claude.json and overrides HOME
// so tests don't touch the real config.
func setupFakeConfig(t *testing.T) (cleanup func()) {
	t.Helper()
	tmp := t.TempDir()
	claudeDir := filepath.Join(tmp, ".claude-work")
	os.MkdirAll(claudeDir, 0o755)
	os.WriteFile(filepath.Join(claudeDir, ".claude.json"), []byte(`{"mcpServers":{}}`), 0o644)
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	return func() { os.Setenv("HOME", origHome) }
}

func readMcpServers(t *testing.T) map[string]any {
	t.Helper()
	data, err := os.ReadFile(configPath())
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	json.Unmarshal(data, &raw)
	servers, _ := raw["mcpServers"].(map[string]any)
	return servers
}

func TestRegister_WritesJSON(t *testing.T) {
	cleanup := setupFakeConfig(t)
	defer cleanup()

	cfg := &registry.ServerConfig{
		Protocol:     "http",
		Runtime:      "docker",
		Port:         9894,
		EndpointPath: "/mcp",
	}
	if err := Register("airflow", cfg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	servers := readMcpServers(t)
	entry, ok := servers["airflow"].(map[string]any)
	if !ok {
		t.Fatalf("airflow not found in mcpServers: %v", servers)
	}
	if got := entry["url"]; got != "http://localhost:9894/mcp" {
		t.Errorf("url: got %q", got)
	}
	if got := entry["type"]; got != "http" {
		t.Errorf("type: got %q", got)
	}
}

func TestRegister_WithHeaders(t *testing.T) {
	cleanup := setupFakeConfig(t)
	defer cleanup()

	cfg := &registry.ServerConfig{
		Protocol:     "http",
		Runtime:      "docker",
		Port:         9890,
		EndpointPath: "/snowflake-mcp",
		Headers:      []string{"Authorization: Bearer snowflake-internal"},
	}
	if err := Register("snowflake", cfg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	servers := readMcpServers(t)
	entry := servers["snowflake"].(map[string]any)
	headers, ok := entry["headers"].(map[string]any)
	if !ok {
		t.Fatalf("headers not found: %v", entry)
	}
	if got := headers["Authorization"]; got != "Bearer snowflake-internal" {
		t.Errorf("Authorization header: got %q", got)
	}
}

func TestDeregister_RemovesEntry(t *testing.T) {
	cleanup := setupFakeConfig(t)
	defer cleanup()

	cfg := &registry.ServerConfig{
		Protocol:     "http",
		Runtime:      "docker",
		Port:         9894,
		EndpointPath: "/mcp",
	}
	Register("airflow", cfg)

	if err := Deregister("airflow", cfg); err != nil {
		t.Fatalf("Deregister: %v", err)
	}

	servers := readMcpServers(t)
	if _, ok := servers["airflow"]; ok {
		t.Error("airflow still present after Deregister")
	}
}

func TestRegister_PreservesExisting(t *testing.T) {
	cleanup := setupFakeConfig(t)
	defer cleanup()

	// Register two servers
	cfg1 := &registry.ServerConfig{Protocol: "http", Runtime: "docker", Port: 9890, EndpointPath: "/snowflake-mcp"}
	cfg2 := &registry.ServerConfig{Protocol: "http", Runtime: "docker", Port: 9894, EndpointPath: "/mcp"}
	Register("snowflake", cfg1)
	Register("airflow", cfg2)

	servers := readMcpServers(t)
	if _, ok := servers["snowflake"]; !ok {
		t.Error("snowflake missing after second Register")
	}
	if _, ok := servers["airflow"]; !ok {
		t.Error("airflow missing")
	}
}

func TestRegister_StdioProcess(t *testing.T) {
	cleanup := setupFakeConfig(t)
	defer cleanup()

	cfg := &registry.ServerConfig{
		Protocol:  "stdio",
		Runtime:   "process",
		NativeCmd: "python",
		NativeArgs: []string{"-m", "graph_mcp"},
		Env: []registry.EnvVar{
			{Name: "GRAPH_MCP_DAEMON", Value: "1"},
			{Name: "FASTMCP_LOG_LEVEL", Value: "ERROR"},
		},
	}
	if err := Register("graph", cfg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	servers := readMcpServers(t)
	entry, ok := servers["graph"].(map[string]any)
	if !ok {
		t.Fatalf("graph not found in mcpServers: %v", servers)
	}
	if got := entry["type"]; got != "stdio" {
		t.Errorf("type: got %q, want \"stdio\"", got)
	}
	if got := entry["command"]; got != "python" {
		t.Errorf("command: got %q, want \"python\"", got)
	}
	if _, hasURL := entry["url"]; hasURL {
		t.Error("stdio entry must not have url field")
	}
	env, _ := entry["env"].(map[string]any)
	if env["GRAPH_MCP_DAEMON"] != "1" {
		t.Errorf("env[GRAPH_MCP_DAEMON]: got %q", env["GRAPH_MCP_DAEMON"])
	}
}

// Legacy tests for registerArgs (no longer used in prod but kept for reference)
func TestRegisterArgs_NoHeaders(t *testing.T) {
	cfg := &registry.ServerConfig{
		Protocol:     "http",
		Runtime:      "docker",
		Port:         9890,
		EndpointPath: "/snowflake-mcp",
	}
	got := registerArgs("snowflake", cfg)
	want := []string{"mcp", "add", "--transport", "http", "--scope", "user", "snowflake", "http://localhost:9890/snowflake-mcp"}
	if len(got) != len(want) {
		t.Fatalf("args length: got %d %v, want %d %v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("args[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}
