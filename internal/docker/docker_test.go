package docker_test

import (
	"os"
	"testing"

	"github.com/paperworlds/textserve/internal/docker"
	"github.com/paperworlds/textserve/internal/registry"
)

func TestResolveEnv_StaticValue(t *testing.T) {
	cfg := &registry.ServerConfig{
		Env: []registry.EnvVar{
			{Name: "FOO", Value: "bar"},
			{Name: "BAZ", Value: "qux"},
		},
	}
	got, err := docker.ResolveEnv("test", cfg)
	if err != nil {
		t.Fatalf("ResolveEnv: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len: got %d, want 2", len(got))
	}
	if got[0] != "FOO=bar" {
		t.Errorf("got[0] = %q, want FOO=bar", got[0])
	}
	if got[1] != "BAZ=qux" {
		t.Errorf("got[1] = %q, want BAZ=qux", got[1])
	}
}

func TestResolveEnv_ValueTemplate(t *testing.T) {
	cfg := &registry.ServerConfig{
		Env: []registry.EnvVar{
			{Name: "BASE_URL", Value: "https://example.com"},
			{Name: "FULL_URL", ValueTemplate: "${BASE_URL}/api"},
		},
	}
	got, err := docker.ResolveEnv("test", cfg)
	if err != nil {
		t.Fatalf("ResolveEnv: %v", err)
	}
	want := "FULL_URL=https://example.com/api"
	if got[1] != want {
		t.Errorf("got[1] = %q, want %q", got[1], want)
	}
}

func TestResolveEnv_Empty(t *testing.T) {
	cfg := &registry.ServerConfig{}
	got, err := docker.ResolveEnv("test", cfg)
	if err != nil {
		t.Fatalf("ResolveEnv: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}

func TestResolveVolumes_HomeExpansion(t *testing.T) {
	home := os.Getenv("HOME")
	if home == "" {
		t.Skip("HOME not set")
	}
	cfg := &registry.ServerConfig{
		Volumes: []registry.Volume{
			{Host: "${HOME}/.mcp/config.yaml", Container: "/app/config.yaml", Readonly: true},
		},
	}
	got, err := docker.ResolveVolumes(cfg)
	if err != nil {
		t.Fatalf("ResolveVolumes: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len: got %d, want 1", len(got))
	}
	want := home + "/.mcp/config.yaml:/app/config.yaml:ro"
	if got[0] != want {
		t.Errorf("got %q, want %q", got[0], want)
	}
}

func TestResolveVolumes_NoReadonly(t *testing.T) {
	cfg := &registry.ServerConfig{
		Volumes: []registry.Volume{
			{Host: "/tmp/data", Container: "/data"},
		},
	}
	got, err := docker.ResolveVolumes(cfg)
	if err != nil {
		t.Fatalf("ResolveVolumes: %v", err)
	}
	want := "/tmp/data:/data"
	if got[0] != want {
		t.Errorf("got %q, want %q", got[0], want)
	}
}
