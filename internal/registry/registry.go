package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// EnvVar defines a single environment variable with its source.
type EnvVar struct {
	Name          string `yaml:"name"`
	Value         string `yaml:"value,omitempty"`
	ValueTemplate string `yaml:"value_template,omitempty"`
	Op            string `yaml:"op,omitempty"`
	Cache         string `yaml:"cache,omitempty"`
	CacheFile     string `yaml:"cache_file,omitempty"`
}

// Volume defines a host→container volume mount.
type Volume struct {
	Host            string `yaml:"host"`
	Container       string `yaml:"container"`
	Readonly        bool   `yaml:"readonly,omitempty"`
	ResolveSymlinks bool   `yaml:"resolve_symlinks,omitempty"`
}

// Dep defines a prerequisite check.
type Dep struct {
	Cmd  string `yaml:"cmd"`
	Hint string `yaml:"hint"`
}

// Health defines the health-check configuration.
type Health struct {
	Endpoint string `yaml:"endpoint,omitempty"`
	Probe    string `yaml:"probe,omitempty"`
	Timeout  int    `yaml:"timeout,omitempty"`
}

// ServerConfig is the full per-server configuration (servers/<name>/server.yaml).
type ServerConfig struct {
	Image         string   `yaml:"image,omitempty"`
	Transport     string   `yaml:"transport"`
	Port          int      `yaml:"port,omitempty"`
	ContainerPort int      `yaml:"container_port,omitempty"`
	EndpointPath  string   `yaml:"endpoint_path,omitempty"`
	Tags          []string `yaml:"tags,omitempty"`
	Network       string   `yaml:"network,omitempty"`
	Env           []EnvVar `yaml:"env,omitempty"`
	Volumes       []Volume `yaml:"volumes,omitempty"`
	ExtraArgs     []string `yaml:"extra_args,omitempty"`
	PreStart      string   `yaml:"pre_start,omitempty"`
	PidFile       string   `yaml:"pid_file,omitempty"`
	NativeCmd     string   `yaml:"native_cmd,omitempty"`
	NativeArgs    []string `yaml:"native_args,omitempty"`
	NativeVenv    string   `yaml:"native_venv,omitempty"`
	Deps          []Dep    `yaml:"deps,omitempty"`
	Health        Health   `yaml:"health,omitempty"`
	ManagedBy     string   `yaml:"managed_by,omitempty"`
}

// RegistryEntry is a single entry in registry.yaml.
type RegistryEntry struct {
	Image         string   `yaml:"image,omitempty"`
	Transport     string   `yaml:"transport"`
	Port          int      `yaml:"port,omitempty"`
	ContainerPort int      `yaml:"container_port,omitempty"`
	EndpointPath  string   `yaml:"endpoint_path,omitempty"`
	Tags          []string `yaml:"tags,omitempty"`
	Deps          []Dep    `yaml:"deps,omitempty"`
	Health        Health   `yaml:"health,omitempty"`
	ManagedBy     string   `yaml:"managed_by,omitempty"`
}

// FleetRegistry is the top-level registry.yaml structure.
type FleetRegistry struct {
	Servers map[string]RegistryEntry `yaml:"servers"`
}

// Load parses registry.yaml at the given path.
func Load(path string) (*FleetRegistry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read registry: %w", err)
	}
	var r FleetRegistry
	if err := yaml.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}
	return &r, nil
}

// LoadServer parses servers/<name>/server.yaml relative to repoRoot.
func LoadServer(repoRoot, name string) (*ServerConfig, error) {
	path := filepath.Join(repoRoot, "servers", name, "server.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read server config %q: %w", name, err)
	}
	var sc ServerConfig
	if err := yaml.Unmarshal(data, &sc); err != nil {
		return nil, fmt.Errorf("parse server config %q: %w", name, err)
	}
	return &sc, nil
}

// ListNames returns all server names sorted alphabetically.
func (r *FleetRegistry) ListNames() []string {
	names := make([]string, 0, len(r.Servers))
	for name := range r.Servers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// FilterByTag returns server names that include the given tag.
func (r *FleetRegistry) FilterByTag(tag string) []string {
	var result []string
	for name, entry := range r.Servers {
		for _, t := range entry.Tags {
			if t == tag {
				result = append(result, name)
				break
			}
		}
	}
	sort.Strings(result)
	return result
}
