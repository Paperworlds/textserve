package textaccounts

import (
	"os"
	"path/filepath"
	"testing"
)

func writeProfilesYAML(t *testing.T, dir, content string) string {
	t.Helper()
	taDir := filepath.Join(dir, ".textaccounts")
	if err := os.MkdirAll(taDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(taDir, "profiles.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const sampleYAML = `
version: '1.0'
active: work

profiles:
  work:
    path: /Users/test/.claude-work
    description: day job
  personal:
    path: /Users/test/.claude-personal
    description: hobby projects
    aliases:
      - p
`

func TestResolveProfile_Direct(t *testing.T) {
	tmp := t.TempDir()
	writeProfilesYAML(t, tmp, sampleYAML)
	t.Setenv("HOME", tmp)

	path, err := ResolveProfile("work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/Users/test/.claude-work" {
		t.Errorf("got %q, want /Users/test/.claude-work", path)
	}
}

func TestResolveProfile_Alias(t *testing.T) {
	tmp := t.TempDir()
	writeProfilesYAML(t, tmp, sampleYAML)
	t.Setenv("HOME", tmp)

	path, err := ResolveProfile("p")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/Users/test/.claude-personal" {
		t.Errorf("got %q, want /Users/test/.claude-personal", path)
	}
}

func TestResolveProfile_NotFound(t *testing.T) {
	tmp := t.TempDir()
	writeProfilesYAML(t, tmp, sampleYAML)
	t.Setenv("HOME", tmp)

	_, err := ResolveProfile("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown profile")
	}
}

func TestResolveProfile_NoFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp) // no profiles.yaml written

	_, err := ResolveProfile("work")
	if err == nil {
		t.Fatal("expected error when profiles.yaml absent")
	}
}

func TestExpandPath_Tilde(t *testing.T) {
	t.Setenv("HOME", "/Users/test")
	got := expandPath("~/.claude-work")
	if got != "/Users/test/.claude-work" {
		t.Errorf("got %q", got)
	}
}

func TestAvailable_True(t *testing.T) {
	tmp := t.TempDir()
	writeProfilesYAML(t, tmp, sampleYAML)
	t.Setenv("HOME", tmp)

	if !Available() {
		t.Error("expected Available()=true")
	}
}

func TestAvailable_False(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	if Available() {
		t.Error("expected Available()=false when no profiles.yaml")
	}
}
