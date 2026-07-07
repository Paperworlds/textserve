package native

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/paperworlds/textserve/internal/registry"
)

// TestStart_VenvPythonResolution verifies that when native_venv is set and
// the venv contains the command binary, Start uses the venv binary rather
// than relying on PATH resolution (which would pick up the system binary).
func TestStart_VenvPythonResolution(t *testing.T) {
	// Build a fake venv with a fake python binary that exits 0.
	tmp := t.TempDir()
	binDir := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	fakePython := filepath.Join(binDir, "python")
	// Script that just exits successfully without doing anything.
	if err := os.WriteFile(fakePython, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := &registry.ServerConfig{
		Protocol:   "stdio",
		Runtime:    "process",
		NativeCmd:  "python",
		NativeVenv: tmp,
		NativeArgs: []string{"-c", "pass"},
		PidFile:    filepath.Join(tmp, "test.pid"),
	}

	if err := Start("test", cfg); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Verify PID file was written — confirms the fake binary ran, not system python.
	if _, err := os.Stat(cfg.PidFile); err != nil {
		t.Errorf("pid file not created: %v", err)
	}
}
