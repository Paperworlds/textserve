// Package localconfig loads the user-local textserve config at
// ~/.config/textserve/config.yaml. This file is never committed — it holds
// machine-specific settings such as 1Password reference paths for env vars.
package localconfig

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Path returns the canonical config file path.
func Path() string {
	home := os.Getenv("HOME")
	return filepath.Join(home, ".config", "textserve", "config.yaml")
}

// ServerOverride holds per-server local overrides.
type ServerOverride struct {
	// Env maps env var name to a value or op:// reference path.
	// Example: SENTRY_AUTH_TOKEN: "op://Private/Sentry MCP/SENTRY_AUTH_TOKEN"
	Env map[string]string `yaml:"env"`
}

// LocalConfig is the top-level structure of ~/.config/textserve/config.yaml.
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

// EnvFor returns the local env overrides for the named server (empty map if none).
func (c *LocalConfig) EnvFor(name string) map[string]string {
	if ov, ok := c.Servers[name]; ok && ov.Env != nil {
		return ov.Env
	}
	return map[string]string{}
}
