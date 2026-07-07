// Package localconfig loads the user-local textserve config at
// ~/.local/paperworlds/textserve/local.yaml. This file is never committed —
// it holds machine-specific settings such as 1Password reference paths for
// env vars.
package localconfig

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Path returns the canonical config file path.
func Path() string {
	home := os.Getenv("HOME")
	return filepath.Join(home, ".local", "paperworlds", "textserve", "local.yaml")
}

// ServerOverride holds per-server local overrides.
type ServerOverride struct {
	// Env maps env var name to a 1Password reference path (resolved by textserve at startup).
	// Example: MY_TOKEN: "op://Vault/Item/field"
	Env map[string]string `yaml:"env"`

	// LiteralEnv maps env var name to a literal string value passed through as-is,
	// without 1Password resolution. Use for values that the server itself needs to pass
	// to op read (e.g. source paths for credential refresh).
	// Example: MY_TOKEN_SOURCE: "op://Vault/Item/field"
	LiteralEnv map[string]string `yaml:"literal_env"`
}

// LocalConfig is the top-level structure of ~/.local/paperworlds/textserve/local.yaml.
type LocalConfig struct {
	Servers map[string]ServerOverride `yaml:"servers"`
}

// Load reads the local config. Returns an empty config (not an error) if the
// file does not exist — the file is optional.
func Load() (*LocalConfig, error) {
	data, err := os.ReadFile(Path())
	if os.IsNotExist(err) {
		return &LocalConfig{Servers: map[string]ServerOverride{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var cfg LocalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Servers == nil {
		cfg.Servers = map[string]ServerOverride{}
	}
	return &cfg, nil
}

// EnvFor returns the op:// env overrides (resolved at startup) for the named server.
func (c *LocalConfig) EnvFor(name string) map[string]string {
	if ov, ok := c.Servers[name]; ok && ov.Env != nil {
		return ov.Env
	}
	return map[string]string{}
}

// LiteralEnvFor returns the literal env values (passed through as-is) for the named server.
func (c *LocalConfig) LiteralEnvFor(name string) map[string]string {
	if ov, ok := c.Servers[name]; ok && ov.LiteralEnv != nil {
		return ov.LiteralEnv
	}
	return map[string]string{}
}
