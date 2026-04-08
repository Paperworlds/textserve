package op

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Read executes `op read <uri>` and returns the trimmed value.
func Read(uri string) (string, error) {
	out, err := exec.Command("op", "read", uri).Output()
	if err != nil {
		return "", fmt.Errorf("op read %q: %w", uri, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Cached returns the value for service/field from ~/.cache/mcp-<service>/<field>,
// populating it from 1Password via opURI if absent.
func Cached(service, field, opURI string) (string, error) {
	cacheDir := filepath.Join(os.Getenv("HOME"), ".cache", "mcp-"+service)
	cacheFile := filepath.Join(cacheDir, field)

	if data, err := os.ReadFile(cacheFile); err == nil {
		return strings.TrimSpace(string(data)), nil
	}

	value, err := Read(opURI)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return "", fmt.Errorf("create cache dir: %w", err)
	}
	if err := os.WriteFile(cacheFile, []byte(value), 0o600); err != nil {
		return "", fmt.Errorf("write cache: %w", err)
	}
	return value, nil
}

// CacheFileRead returns the value from ~/.cache/<cacheRelPath> (pre-populated externally).
func CacheFileRead(cacheRelPath string) (string, error) {
	path := filepath.Join(os.Getenv("HOME"), ".cache", cacheRelPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("cache_file read %q: %w", cacheRelPath, err)
	}
	return strings.TrimSpace(string(data)), nil
}
