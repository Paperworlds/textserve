package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/pdonorio/mcp-fleet/internal/health"
	"github.com/pdonorio/mcp-fleet/internal/registry"
)

// StatusSummary is the JSON structure written to ~/.files/states/mcp-fleet.json.
type StatusSummary struct {
	UpdatedAt string   `json:"updated_at"`
	Healthy   int      `json:"healthy"`
	Total     int      `json:"total"`
	Unhealthy []string `json:"unhealthy"`
}

// serverRow holds per-server status for the table display.
type serverRow struct {
	name      string
	transport string
	port      string
	status    string
	uptime    string
}

// ProbeFunc is the health probe signature used by the status command.
// It can be overridden in tests.
var ProbeFunc = func(name string, cfg *registry.ServerConfig) (string, error) {
	return health.Probe(name, cfg)
}

func newStatusCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of all MCP servers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fleet, repoRoot, err := loadFleet()
			if err != nil {
				return err
			}

			names := fleet.ListNames()
			rows := make([]serverRow, len(names))
			var mu sync.Mutex
			var wg sync.WaitGroup

			for i, n := range names {
				wg.Add(1)
				go func(idx int, name string) {
					defer wg.Done()
					entry := fleet.Servers[name]
					cfg := serverConfig(repoRoot, name, entry)

					status, _ := ProbeFunc(name, cfg)
					if status == "" {
						status = "unknown"
					}

					portStr := "-"
					if cfg.Port > 0 {
						portStr = strconv.Itoa(cfg.Port)
					}

					row := serverRow{
						name:      name,
						transport: cfg.Transport,
						port:      portStr,
						status:    status,
						uptime:    uptimeFor(name, cfg),
					}
					mu.Lock()
					rows[idx] = row
					mu.Unlock()
				}(i, n)
			}
			wg.Wait()

			// Sort rows by name (names slice is already sorted, rows indexed to match).
			sort.Slice(rows, func(i, j int) bool { return rows[i].name < rows[j].name })

			summary := buildSummary(rows)

			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				if err := enc.Encode(summary); err != nil {
					return err
				}
			} else {
				printStatusTable(cmd, rows)
			}

			return writeSummaryFile(summary)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON summary")
	return cmd
}

func buildSummary(rows []serverRow) StatusSummary {
	var unhealthy []string
	healthyCount := 0
	for _, r := range rows {
		if r.status == "healthy" {
			healthyCount++
		} else if r.status == "unhealthy" {
			unhealthy = append(unhealthy, r.name)
		}
	}
	if unhealthy == nil {
		unhealthy = []string{}
	}
	return StatusSummary{
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Healthy:   healthyCount,
		Total:     len(rows),
		Unhealthy: unhealthy,
	}
}

func printStatusTable(cmd *cobra.Command, rows []serverRow) {
	tw := tablewriter.NewWriter(cmd.OutOrStdout())
	tw.Header("NAME", "TRANSPORT", "PORT", "STATUS", "UPTIME")
	for _, r := range rows {
		_ = tw.Append([]string{r.name, r.transport, r.port, r.status, r.uptime})
	}
	_ = tw.Render()
}

func writeSummaryFile(s StatusSummary) error {
	dir := filepath.Join(os.Getenv("HOME"), ".files", "states")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create states dir: %w", err)
	}
	path := filepath.Join(dir, "mcp-fleet.json")
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// uptimeFor returns a human-readable uptime string for the given server.
func uptimeFor(name string, cfg *registry.ServerConfig) string {
	switch cfg.Transport {
	case "stdio":
		return "-"
	case "native":
		return nativeUptime(cfg)
	default:
		return dockerUptime(name)
	}
}

func dockerUptime(name string) string {
	out, err := exec.Command("docker", "inspect",
		"--format", "{{.State.StartedAt}}",
		"mcp-"+name).Output()
	if err != nil {
		return "-"
	}
	startedAt := strings.TrimSpace(string(out))
	t, err := time.Parse(time.RFC3339Nano, startedAt)
	if err != nil {
		return "-"
	}
	return formatDuration(time.Since(t))
}

func nativeUptime(cfg *registry.ServerConfig) string {
	if cfg.PidFile == "" {
		return "-"
	}
	data, err := os.ReadFile(cfg.PidFile)
	if err != nil {
		return "-"
	}
	pid := strings.TrimSpace(string(data))
	// Use ps on macOS/Linux to get process start time.
	out, err := exec.Command("ps", "-o", "lstart=", "-p", pid).Output()
	if err != nil {
		return "-"
	}
	// ps lstart format: "Mon Apr  7 14:00:00 2026"
	t, err := time.Parse("Mon Jan  2 15:04:05 2006", strings.TrimSpace(string(out)))
	if err != nil {
		// Try alternate format with double-space for single-digit days
		t, err = time.Parse("Mon Jan _2 15:04:05 2006", strings.TrimSpace(string(out)))
		if err != nil {
			return "-"
		}
	}
	return formatDuration(time.Since(t))
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	if d < 0 {
		return "-"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 24 {
		days := h / 24
		h = h % 24
		return fmt.Sprintf("%dd%dh%dm", days, h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%dh%dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
