package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/paperworlds/textserve/internal/registry"
)

// mockProbe returns the provided status string and no error.
func mockProbe(status string) func(string, *registry.ServerConfig) (string, error) {
	return func(name string, cfg *registry.ServerConfig) (string, error) {
		return status, nil
	}
}

func TestBuildSummary_AllHealthy(t *testing.T) {
	rows := []serverRow{
		{name: "jenkins", status: "healthy"},
		{name: "grafana", status: "healthy"},
	}
	s := buildSummary(rows)
	if s.Healthy != 2 {
		t.Errorf("expected Healthy=2, got %d", s.Healthy)
	}
	if s.Total != 2 {
		t.Errorf("expected Total=2, got %d", s.Total)
	}
	if len(s.Unhealthy) != 0 {
		t.Errorf("expected no unhealthy servers, got %v", s.Unhealthy)
	}
}

func TestBuildSummary_SomeUnhealthy(t *testing.T) {
	rows := []serverRow{
		{name: "jenkins", status: "healthy"},
		{name: "airbyte", status: "unhealthy"},
		{name: "airflow", status: "unhealthy"},
	}
	s := buildSummary(rows)
	if s.Healthy != 1 {
		t.Errorf("expected Healthy=1, got %d", s.Healthy)
	}
	if s.Total != 3 {
		t.Errorf("expected Total=3, got %d", s.Total)
	}
	if len(s.Unhealthy) != 2 {
		t.Errorf("expected 2 unhealthy, got %v", s.Unhealthy)
	}
}

func TestStatusJSONSchema(t *testing.T) {
	rows := []serverRow{
		{name: "jenkins", mode: "docker", port: "9887", status: "healthy", uptime: "1h"},
		{name: "airbyte", mode: "docker", port: "9893", status: "unhealthy", uptime: "-"},
	}
	s := buildSummary(rows)

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	for _, key := range []string{"updated_at", "healthy", "total", "unhealthy"} {
		if _, ok := out[key]; !ok {
			t.Errorf("missing key %q in JSON output", key)
		}
	}
}

func TestWriteSummaryFile(t *testing.T) {
	// Use a temp dir as $HOME so we don't pollute the real one.
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	s := StatusSummary{
		UpdatedAt: "2026-04-08T14:00:00Z",
		Healthy:   5,
		Total:     10,
		Unhealthy: []string{"airbyte"},
	}
	if err := writeSummaryFile(s); err != nil {
		t.Fatalf("writeSummaryFile: %v", err)
	}

	expectedPath := filepath.Join(tmpHome, ".files", "states", "mcp-fleet.json")
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("state file not written at %s: %v", expectedPath, err)
	}

	var parsed StatusSummary
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("parse written file: %v", err)
	}
	if parsed.Total != 10 {
		t.Errorf("expected Total=10 in file, got %d", parsed.Total)
	}
}

func TestPrintStatusTable_NoError(t *testing.T) {
	rows := []serverRow{
		{name: "jenkins", mode: "docker", port: "9887", status: "healthy", registered: true, uptime: "1h"},
	}
	cmd := buildRoot()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	printStatusTable(cmd, rows)
	if buf.Len() == 0 {
		t.Error("expected table output, got empty")
	}
}
