package claude

import (
	"testing"

	"github.com/pdonorio/mcp-fleet/internal/registry"
)

func TestRegisterArgs_NoHeaders(t *testing.T) {
	cfg := &registry.ServerConfig{
		Transport:    "http",
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

func TestRegisterArgs_WithHeaders(t *testing.T) {
	cfg := &registry.ServerConfig{
		Transport:    "http",
		Port:         9890,
		EndpointPath: "/snowflake-mcp",
		Headers:      []string{"Authorization: Bearer snowflake-internal"},
	}
	got := registerArgs("snowflake", cfg)
	want := []string{
		"mcp", "add", "--transport", "http", "--scope", "user",
		"--header", "Authorization: Bearer snowflake-internal",
		"snowflake", "http://localhost:9890/snowflake-mcp",
	}
	if len(got) != len(want) {
		t.Fatalf("args length: got %d %v, want %d %v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("args[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestRegisterArgs_MultipleHeaders(t *testing.T) {
	cfg := &registry.ServerConfig{
		Transport:    "http",
		Port:         9897,
		EndpointPath: "/mcp",
		Headers:      []string{"Authorization: Bearer tok1", "X-Custom: val"},
	}
	got := registerArgs("datadog", cfg)
	// Check --header pairs appear in order
	headerIdx := -1
	for i, a := range got {
		if a == "--header" {
			headerIdx = i
			break
		}
	}
	if headerIdx == -1 {
		t.Fatal("no --header flag found in args")
	}
	if got[headerIdx+1] != "Authorization: Bearer tok1" {
		t.Errorf("first header value: got %q", got[headerIdx+1])
	}
	if got[headerIdx+3] != "X-Custom: val" {
		t.Errorf("second header value: got %q", got[headerIdx+3])
	}
}
