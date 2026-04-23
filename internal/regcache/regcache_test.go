package regcache_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/paperworlds/textserve/internal/regcache"
)

func TestHashRoundTrip(t *testing.T) {
	dir := t.TempDir()
	content := []byte("protocol: stdio\nruntime: claude\n")
	f := filepath.Join(dir, "server.yaml")
	if err := os.WriteFile(f, content, 0o644); err != nil {
		t.Fatal(err)
	}
	h1, err := regcache.ComputeServerYAMLHash(dir, "test", nil)
	if err != nil {
		t.Fatalf("hash 1: %v", err)
	}
	h2, err := regcache.ComputeServerYAMLHash(dir, "test", nil)
	if err != nil {
		t.Fatalf("hash 2: %v", err)
	}
	if h1 != h2 {
		t.Errorf("hashes differ: %q vs %q", h1, h2)
	}
}

func TestWriteReadHash(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := regcache.WriteHash("myserver", "abc123"); err != nil {
		t.Fatalf("WriteHash: %v", err)
	}
	got, err := regcache.ReadStoredHash("myserver")
	if err != nil {
		t.Fatalf("ReadStoredHash: %v", err)
	}
	if got != "abc123" {
		t.Errorf("got %q, want abc123", got)
	}
}

func TestReadStoredHash_Absent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	got, err := regcache.ReadStoredHash("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestHashChanges(t *testing.T) {
	dir := t.TempDir()
	serverDir := filepath.Join(dir, "servers", "test")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(serverDir, "server.yaml")
	if err := os.WriteFile(f, []byte("protocol: stdio\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	h1, _ := regcache.ComputeServerYAMLHash(dir, "test", nil)
	if err := os.WriteFile(f, []byte("protocol: stdio\nruntime: claude\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	h2, _ := regcache.ComputeServerYAMLHash(dir, "test", nil)
	if h1 == h2 {
		t.Error("expected different hashes after file change")
	}
}

func TestHashFallbackToEntryYAML(t *testing.T) {
	dir := t.TempDir() // no server.yaml inside
	fallback := []byte("runtime: docker\n")
	h, err := regcache.ComputeServerYAMLHash(dir, "missing", fallback)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h == "" {
		t.Error("expected non-empty hash from fallback")
	}
}
