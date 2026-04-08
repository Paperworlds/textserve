package health_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/pdonorio/mcp-fleet/internal/health"
	"github.com/pdonorio/mcp-fleet/internal/registry"
)

// serverPort extracts the numeric port from a URL like "http://127.0.0.1:PORT".
func serverPort(t *testing.T, rawURL string) int {
	t.Helper()
	parts := strings.Split(rawURL, ":")
	if len(parts) < 3 {
		t.Fatalf("unexpected URL format: %s", rawURL)
	}
	port, err := strconv.Atoi(parts[2])
	if err != nil {
		t.Fatalf("parse port from %q: %v", rawURL, err)
	}
	return port
}

func TestProbeHTTP_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &registry.ServerConfig{
		Port: serverPort(t, srv.URL),
		Health: registry.Health{
			Endpoint: "/",
			Timeout:  5,
		},
	}
	if err := health.ProbeHTTP("test", cfg); err != nil {
		t.Fatalf("expected healthy, got: %v", err)
	}
}

func TestProbeHTTP_500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := &registry.ServerConfig{
		Port: serverPort(t, srv.URL),
		Health: registry.Health{
			Endpoint: "/",
			Timeout:  5,
		},
	}
	if err := health.ProbeHTTP("test", cfg); err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestProbeHTTP_Unreachable(t *testing.T) {
	cfg := &registry.ServerConfig{
		Port: 19988, // unused port
		Health: registry.Health{
			Endpoint: "/health",
			Timeout:  1,
		},
	}
	if err := health.ProbeHTTP("test", cfg); err == nil {
		t.Fatal("expected error for unreachable port")
	}
}

func TestProbePID_CurrentProcess(t *testing.T) {
	pid := os.Getpid()
	f, err := os.CreateTemp("", "mcpf-test-pid-*.pid")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	fmt.Fprintf(f, "%d\n", pid)
	f.Close()

	cfg := &registry.ServerConfig{PidFile: f.Name()}
	if err := health.ProbePID(cfg); err != nil {
		t.Fatalf("expected current process to be alive: %v", err)
	}
}

func TestProbePID_DeadProcess(t *testing.T) {
	f, err := os.CreateTemp("", "mcpf-test-pid-*.pid")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	fmt.Fprintf(f, "99999\n")
	f.Close()

	cfg := &registry.ServerConfig{PidFile: f.Name()}
	if err := health.ProbePID(cfg); err == nil {
		t.Fatal("expected error for dead/nonexistent process")
	}
}
