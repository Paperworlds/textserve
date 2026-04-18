package health

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/paperworlds/textserve/internal/registry"
)

// Server status values returned by Probe and used by callers across the stack.
const (
	StatusHealthy   = "healthy"
	StatusUnhealthy = "unhealthy"
	StatusUnknown   = "unknown"
	StatusStopped   = "stopped"
)

// ProbeHTTP checks that the HTTP health endpoint returns a 2xx status code.
func ProbeHTTP(name string, cfg *registry.ServerConfig) error {
	endpoint := cfg.Health.Endpoint
	if endpoint == "" {
		endpoint = "/health"
	}
	timeout := cfg.Health.Timeout
	if timeout <= 0 {
		timeout = 5
	}
	url := fmt.Sprintf("http://localhost:%d%s", cfg.Port, endpoint)
	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

// ProbePID checks that the process recorded in cfg.PidFile is still alive.
func ProbePID(cfg *registry.ServerConfig) error {
	if cfg.PidFile == "" {
		return fmt.Errorf("no pid_file configured")
	}
	data, err := os.ReadFile(cfg.PidFile)
	if err != nil {
		return fmt.Errorf("read pid file: %w", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return fmt.Errorf("parse pid: %w", err)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process %d: %w", pid, err)
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return fmt.Errorf("process %d not alive: %w", pid, err)
	}
	return nil
}

// ProbeToolList probes an stdio MCP server via claude mcp get-tools.
// Returns a "probe not implemented" error when the claude CLI is unavailable.
func ProbeToolList(name string) error {
	out, err := exec.Command(registry.RuntimeClaude, "mcp", "get-tools", name).Output()
	if err != nil {
		return fmt.Errorf("probe not implemented: %w", err)
	}
	if len(strings.TrimSpace(string(out))) == 0 {
		return fmt.Errorf("no tools returned for %s", name)
	}
	return nil
}

// ProbeTCP checks that the server's port accepts TCP connections.
// Useful for servers that have no HTTP health endpoint.
func ProbeTCP(cfg *registry.ServerConfig) error {
	timeout := cfg.Health.Timeout
	if timeout <= 0 {
		timeout = 5
	}
	addr := fmt.Sprintf("localhost:%d", cfg.Port)
	conn, err := net.DialTimeout("tcp", addr, time.Duration(timeout)*time.Second)
	if err != nil {
		return fmt.Errorf("tcp connect: %w", err)
	}
	conn.Close()
	return nil
}

// Probe dispatches to the appropriate probe based on runtime and config.
// Returns ("healthy", nil), ("unhealthy", err), or ("unknown", nil).
func Probe(name string, cfg *registry.ServerConfig) (string, error) {
	switch cfg.Runtime {
	case registry.RuntimeDocker:
		if cfg.Health.Probe == "tcp" && cfg.Port > 0 {
			if err := ProbeTCP(cfg); err != nil {
				return StatusUnhealthy, err
			}
			return StatusHealthy, nil
		}
		if cfg.Port > 0 {
			if err := ProbeHTTP(name, cfg); err != nil {
				return StatusUnhealthy, err
			}
			return StatusHealthy, nil
		}
		return StatusUnknown, nil

	case registry.RuntimeProcess:
		if cfg.Health.Probe == "tcp" && cfg.Port > 0 {
			if err := ProbeTCP(cfg); err != nil {
				return StatusUnhealthy, err
			}
			return StatusHealthy, nil
		}
		if err := ProbePID(cfg); err != nil {
			return StatusUnhealthy, err
		}
		return StatusHealthy, nil

	case registry.RuntimeClaude:
		if cfg.Health.Probe == "tool-list" {
			err := ProbeToolList(name)
			if err != nil && strings.Contains(err.Error(), "probe not implemented") {
				return StatusUnknown, nil
			}
			if err != nil {
				return StatusUnhealthy, err
			}
			return StatusHealthy, nil
		}
		return StatusUnknown, nil

	default:
		return StatusUnknown, nil
	}
}
