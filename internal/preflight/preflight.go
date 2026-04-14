package preflight

import (
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/pdonorio/mcp-fleet/internal/deps"
	"github.com/pdonorio/mcp-fleet/internal/health"
	"github.com/pdonorio/mcp-fleet/internal/registry"
)

// ServerResult holds the preflight result for a single server.
type ServerResult struct {
	Status string `json:"status"`
	Port   int    `json:"port,omitempty"`
	Tools  int    `json:"tools,omitempty"`
	Error  string `json:"error,omitempty"`
}

// Report is the full preflight report. JSON schema is fixed — knowledge-harvest depends on it.
type Report struct {
	Timestamp     time.Time               `json:"timestamp"`
	RequestedTags []string                `json:"requested_tags"`
	Servers       map[string]ServerResult `json:"servers"`
	Ready         bool                    `json:"ready"`
	Blocking      []string                `json:"blocking"`
}

// Run executes preflight checks for all servers matching any of the given tags.
// If tags is empty, all servers are checked.
func Run(tags []string, repoRoot string) (*Report, error) {
	fleet, err := registry.Load(filepath.Join(repoRoot, "registry.yaml"))
	if err != nil {
		return nil, fmt.Errorf("load registry: %w", err)
	}

	// Collect matching server names (union of tags).
	var names []string
	if len(tags) == 0 {
		names = fleet.ListNames()
	} else {
		seen := map[string]bool{}
		for _, tag := range tags {
			for _, n := range fleet.FilterByTag(tag) {
				if !seen[n] {
					seen[n] = true
					names = append(names, n)
				}
			}
		}
		sort.Strings(names)
	}

	results := make(map[string]ServerResult, len(names))
	var blocking []string

	for _, name := range names {
		entry := fleet.Servers[name]
		sc, scErr := registry.LoadServer(repoRoot, name)
		if scErr != nil {
			sc = &registry.ServerConfig{
				Image:         entry.Image,
				Protocol:      entry.Protocol,
				Runtime:       entry.Runtime,
				Port:          entry.Port,
				ContainerPort: entry.ContainerPort,
				EndpointPath:  entry.EndpointPath,
				Tags:          entry.Tags,
				Deps:          entry.Deps,
				Health:        entry.Health,
			}
		}

		result := checkServer(name, sc)
		results[name] = result
		if result.Status == "unhealthy" {
			blocking = append(blocking, name)
		}
	}

	if blocking == nil {
		blocking = []string{}
	}
	sort.Strings(blocking)

	ready := len(blocking) == 0
	// "stopped" servers do not block readiness for the purpose of this report.
	// Only "unhealthy" counts as blocking.

	return &Report{
		Timestamp:     time.Now().UTC().Truncate(time.Second),
		RequestedTags: tags,
		Servers:       results,
		Ready:         ready,
		Blocking:      blocking,
	}, nil
}

// checkServer runs dep checks and a health probe for a single server.
func checkServer(name string, sc *registry.ServerConfig) ServerResult {
	// Claude-managed servers: probe via tool-list only, no deps.
	if sc.Runtime == "claude" {
		status, err := health.Probe(name, sc)
		if err != nil {
			return ServerResult{Status: "unhealthy", Error: err.Error()}
		}
		if status == "unknown" {
			return ServerResult{Status: "stopped"}
		}
		return ServerResult{Status: status}
	}

	// Dep checks.
	if err := deps.Check(sc.Deps); err != nil {
		return ServerResult{
			Status: "unhealthy",
			Port:   sc.Port,
			Error:  fmt.Sprintf("dep_failed: %s", err.Error()),
		}
	}

	// Health probe.
	status, err := health.Probe(name, sc)
	if err != nil {
		return ServerResult{Status: "unhealthy", Port: sc.Port, Error: err.Error()}
	}
	switch status {
	case "healthy":
		return ServerResult{Status: "healthy", Port: sc.Port}
	case "unknown":
		return ServerResult{Status: "stopped"}
	default:
		return ServerResult{Status: status, Port: sc.Port}
	}
}
