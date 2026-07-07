package registry_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/paperworlds/textserve/internal/registry"
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
	if len(r.Servers) == 0 {
		t.Error("expected at least one server in registry.yaml")
	}
}

func TestFilterByTag_Gateway(t *testing.T) {
	root := repoRoot()
	r, err := registry.Load(filepath.Join(root, "registry.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := r.FilterByTag("gateway")
	if len(got) == 0 {
		t.Fatalf("FilterByTag(gateway): expected at least one server, got none")
	}
	// every returned server must actually carry the tag
	for _, name := range got {
		entry := r.Servers[name]
		tagged := false
		for _, tag := range entry.Tags {
			if tag == "gateway" {
				tagged = true
			}
		}
		if !tagged {
			t.Errorf("FilterByTag(gateway): %q lacks the gateway tag", name)
		}
	}
}

func TestLoadServer_ToolsAPI(t *testing.T) {
	root := repoRoot()
	sc, err := registry.LoadServer(root, "tools-api")
	if err != nil {
		t.Fatalf("LoadServer: %v", err)
	}
	if sc.Port != 10893 {
		t.Errorf("tools-api port: got %d, want 10893", sc.Port)
	}
	if len(sc.Env) == 0 {
		t.Error("tools-api env: got none, want at least one")
	}
}
