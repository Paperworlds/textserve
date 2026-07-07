// Package togglestate manages the per-host overlay that records which bundles
// have been toggled on/off relative to the registry defaults. The overlay
// lives at $TEXTSERVE_STATE_DIR/bundles.yaml (default ~/.local/textserve/).
package togglestate

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

const overlayFilename = "bundles.yaml"

// Overlay is the on-disk shape of the state file.
type Overlay struct {
	Bundles map[string]bool `yaml:"bundles"`
}

// StateDir returns the directory used for textserve state files.
// Honors $TEXTSERVE_STATE_DIR; otherwise ~/.local/textserve.
func StateDir() (string, error) {
	if d := os.Getenv("TEXTSERVE_STATE_DIR"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "textserve"), nil
}

// OverlayPath returns the full path to the bundles overlay file.
func OverlayPath() (string, error) {
	dir, err := StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, overlayFilename), nil
}

// Load reads the overlay from disk. A missing file yields an empty overlay.
func Load() (*Overlay, error) {
	path, err := OverlayPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Overlay{Bundles: map[string]bool{}}, nil
		}
		return nil, fmt.Errorf("read overlay: %w", err)
	}
	var o Overlay
	if err := yaml.Unmarshal(data, &o); err != nil {
		return nil, fmt.Errorf("parse overlay: %w", err)
	}
	if o.Bundles == nil {
		o.Bundles = map[string]bool{}
	}
	return &o, nil
}

// Save writes the overlay atomically.
func (o *Overlay) Save() error {
	path, err := OverlayPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir state dir: %w", err)
	}
	data, err := yaml.Marshal(o)
	if err != nil {
		return fmt.Errorf("marshal overlay: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write overlay: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename overlay: %w", err)
	}
	return nil
}

// Set records an explicit enabled state for a bundle.
func (o *Overlay) Set(name string, enabled bool) {
	if o.Bundles == nil {
		o.Bundles = map[string]bool{}
	}
	o.Bundles[name] = enabled
}

// Lookup returns the overlay value for a bundle if one is set.
func (o *Overlay) Lookup(name string) (enabled, present bool) {
	v, ok := o.Bundles[name]
	return v, ok
}

// Effective returns the effective enabled state for a bundle: overlay value
// if present, otherwise the registry default.
func (o *Overlay) Effective(name string, registryDefault bool) bool {
	if v, ok := o.Bundles[name]; ok {
		return v
	}
	return registryDefault
}

// Names returns the bundle names recorded in the overlay (sorted).
func (o *Overlay) Names() []string {
	names := make([]string, 0, len(o.Bundles))
	for n := range o.Bundles {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
