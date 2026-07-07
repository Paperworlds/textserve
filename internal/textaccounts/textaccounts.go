package textaccounts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type profileEntry struct {
	Path    string   `yaml:"path"`
	Aliases []string `yaml:"aliases,omitempty"`
}

type profilesFile struct {
	Profiles map[string]profileEntry `yaml:"profiles"`
}

// ProfilesPath returns the canonical path to textaccounts' profiles.yaml.
func ProfilesPath() string {
	return filepath.Join(os.Getenv("HOME"), ".textaccounts", "profiles.yaml")
}

// Available reports whether textaccounts is configured (profiles.yaml exists with at least one profile).
func Available() bool {
	pf, err := load()
	if err != nil {
		return false
	}
	return len(pf.Profiles) > 0
}

// ResolveProfile returns the Claude config directory path for the named profile.
// It resolves aliases and expands "~/". Returns an error if textaccounts is not
// configured or the profile is not found.
func ResolveProfile(name string) (string, error) {
	pf, err := load()
	if err != nil {
		return "", fmt.Errorf("textaccounts not configured (%s): %w", ProfilesPath(), err)
	}

	// Direct name match.
	if entry, ok := pf.Profiles[name]; ok {
		return expandPath(entry.Path), nil
	}

	// Alias scan.
	for _, entry := range pf.Profiles {
		for _, alias := range entry.Aliases {
			if alias == name {
				return expandPath(entry.Path), nil
			}
		}
	}

	return "", fmt.Errorf("textaccounts profile %q not found", name)
}

// ListProfiles returns sorted profile names.
func ListProfiles() ([]string, error) {
	pf, err := load()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(pf.Profiles))
	for n := range pf.Profiles {
		names = append(names, n)
	}
	return names, nil
}

func load() (*profilesFile, error) {
	data, err := os.ReadFile(ProfilesPath())
	if err != nil {
		return nil, err
	}
	var pf profilesFile
	if err := yaml.Unmarshal(data, &pf); err != nil {
		return nil, err
	}
	return &pf, nil
}

func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(os.Getenv("HOME"), p[2:])
	}
	return p
}
