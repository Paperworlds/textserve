package registry_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pdonorio/mcp-fleet/internal/registry"
)

// repoRoot returns the absolute path to the repository root, derived from the
// test file's location (internal/registry/ → ../../).
func repoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func TestLoad_AllServers(t *testing.T) {
	root := repoRoot()
	r, err := registry.Load(filepath.Join(root, "registry.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := 13
	if got := len(r.Servers); got != want {
		t.Errorf("server count: got %d, want %d", got, want)
	}
}

func TestFilterByTag_Docker(t *testing.T) {
	root := repoRoot()
	r, err := registry.Load(filepath.Join(root, "registry.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := r.FilterByTag("docker")
	want := 9
	if len(got) != want {
		t.Errorf("FilterByTag(docker): got %d servers %v, want %d", len(got), got, want)
	}
}

func TestLoadServer_Jenkins(t *testing.T) {
	root := repoRoot()
	sc, err := registry.LoadServer(root, "jenkins")
	if err != nil {
		t.Fatalf("LoadServer: %v", err)
	}
	if sc.Port != 9887 {
		t.Errorf("jenkins port: got %d, want 9887", sc.Port)
	}
	if got := len(sc.Env); got != 3 {
		t.Errorf("jenkins env count: got %d, want 3", got)
	}
}

func TestLoadServer_Snowflake_Headers(t *testing.T) {
	root := repoRoot()
	sc, err := registry.LoadServer(root, "snowflake")
	if err != nil {
		t.Fatalf("LoadServer: %v", err)
	}
	if len(sc.Headers) == 0 {
		t.Fatal("snowflake headers: got none, want at least one")
	}
	want := "Authorization: Bearer snowflake-internal"
	if sc.Headers[0] != want {
		t.Errorf("snowflake headers[0]: got %q, want %q", sc.Headers[0], want)
	}
}
