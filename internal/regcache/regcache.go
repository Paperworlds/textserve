// Package regcache stores per-server registration hashes so that re-registration
// can be skipped when server.yaml has not changed since the last register.
package regcache

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
)

// CacheDir returns the directory used for registration hashes.
func CacheDir() string {
	return filepath.Join(os.Getenv("HOME"), ".cache", "textserve")
}

// HashPath returns the path to the stored hash file for the given server name.
func HashPath(name string) string {
	return filepath.Join(CacheDir(), name+".reg.hash")
}

// ComputeServerYAMLHash returns the SHA-256 hex digest of servers/<name>/server.yaml
// relative to repoRoot. Falls back to hashing entryYAML if the file does not exist.
func ComputeServerYAMLHash(repoRoot, name string, entryYAML []byte) (string, error) {
	path := filepath.Join(repoRoot, "servers", name, "server.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		data = entryYAML
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// ReadStoredHash returns the previously written hash for name, or "" if absent.
func ReadStoredHash(name string) (string, error) {
	data, err := os.ReadFile(HashPath(name))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteHash persists hash as the registration hash for name.
func WriteHash(name, hash string) error {
	if err := os.MkdirAll(CacheDir(), 0o755); err != nil {
		return err
	}
	return os.WriteFile(HashPath(name), []byte(hash), 0o644)
}
