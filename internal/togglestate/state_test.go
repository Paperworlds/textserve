package togglestate

import (
	"os"
	"path/filepath"
	"testing"
)

func setTempStateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("TEXTSERVE_STATE_DIR", dir)
	return dir
}

func TestLoad_MissingFile_ReturnsEmpty(t *testing.T) {
	setTempStateDir(t)
	o, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(o.Bundles) != 0 {
		t.Errorf("expected empty overlay, got %v", o.Bundles)
	}
}

func TestSetAndSaveAndLoadRoundtrip(t *testing.T) {
	dir := setTempStateDir(t)
	o := &Overlay{Bundles: map[string]bool{}}
	o.Set("memory", true)
	o.Set("github", false)
	if err := o.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// file should exist
	if _, err := os.Stat(filepath.Join(dir, "bundles.yaml")); err != nil {
		t.Fatalf("overlay file not written: %v", err)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if v, ok := loaded.Lookup("memory"); !ok || !v {
		t.Errorf("memory: got %v ok=%v, want true", v, ok)
	}
	if v, ok := loaded.Lookup("github"); !ok || v {
		t.Errorf("github: got %v ok=%v, want false", v, ok)
	}
}

func TestEffective_OverlayBeatsDefault(t *testing.T) {
	setTempStateDir(t)
	o := &Overlay{Bundles: map[string]bool{"a": true, "b": false}}

	if !o.Effective("a", false) {
		t.Error("overlay-true should beat default-false")
	}
	if o.Effective("b", true) {
		t.Error("overlay-false should beat default-true")
	}
	if !o.Effective("missing", true) {
		t.Error("missing overlay should fall back to default")
	}
	if o.Effective("missing", false) {
		t.Error("missing overlay should fall back to default")
	}
}
