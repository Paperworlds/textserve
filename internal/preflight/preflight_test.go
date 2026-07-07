package preflight

import (
	"encoding/json"
	"testing"
	"time"
)

func TestReportJSONSchema(t *testing.T) {
	ts, _ := time.Parse(time.RFC3339, "2026-04-08T14:30:00Z")
	report := &Report{
		Timestamp:     ts,
		RequestedTags: []string{"data", "monitoring"},
		Servers: map[string]ServerResult{
			"datadog": {Status: "healthy", Port: 9897},
			"airbyte": {Status: "unhealthy", Error: "dep_failed: kubectl port-forward not active"},
			"airflow": {Status: "stopped"},
		},
		Ready:    false,
		Blocking: []string{"airbyte"},
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Top-level keys must be exactly these five.
	required := []string{"timestamp", "requested_tags", "servers", "ready", "blocking"}
	for _, key := range required {
		if _, ok := raw[key]; !ok {
			t.Errorf("missing top-level key %q", key)
		}
	}
	if len(raw) != len(required) {
		t.Errorf("unexpected extra top-level keys: got %d want %d", len(raw), len(required))
	}

	// ready must be false.
	if raw["ready"].(bool) != false {
		t.Errorf("ready should be false")
	}

	// servers: datadog must have port, no tools (omitted when 0).
	servers := raw["servers"].(map[string]interface{})
	datadog := servers["datadog"].(map[string]interface{})
	if datadog["status"] != "healthy" {
		t.Errorf("datadog status: got %v want healthy", datadog["status"])
	}
	if _, hasTools := datadog["tools"]; hasTools {
		t.Error("tools should be omitted when 0")
	}
	if _, hasError := datadog["error"]; hasError {
		t.Error("error should be omitted when empty")
	}

	// airbyte must have error, no tools.
	airbyte := servers["airbyte"].(map[string]interface{})
	if airbyte["error"] == "" {
		t.Error("airbyte error should not be empty")
	}

	// airflow: stopped, no port, no error.
	airflow := servers["airflow"].(map[string]interface{})
	if airflow["status"] != "stopped" {
		t.Errorf("airflow status: got %v want stopped", airflow["status"])
	}
	if _, hasPort := airflow["port"]; hasPort {
		t.Error("airflow port should be omitted when 0")
	}
}

func TestReportReadyFalseWhenUnhealthy(t *testing.T) {
	report := &Report{
		Timestamp:     time.Now().UTC(),
		RequestedTags: []string{"docker"},
		Servers: map[string]ServerResult{
			"a": {Status: "healthy", Port: 9887},
			"b": {Status: "unhealthy", Error: "connect: refused"},
		},
		Ready:    false,
		Blocking: []string{"b"},
	}
	if report.Ready {
		t.Error("ready should be false when any server is unhealthy")
	}
}

func TestReportReadyTrueAllHealthy(t *testing.T) {
	report := &Report{
		Timestamp:     time.Now().UTC(),
		RequestedTags: []string{"ci"},
		Servers: map[string]ServerResult{
			"jenkins": {Status: "healthy", Port: 9887},
		},
		Ready:    true,
		Blocking: []string{},
	}
	if !report.Ready {
		t.Error("ready should be true when all servers are healthy")
	}
	if len(report.Blocking) != 0 {
		t.Errorf("blocking should be empty, got %v", report.Blocking)
	}
}

func TestBlockingContainsCorrectServers(t *testing.T) {
	report := &Report{
		Timestamp:     time.Now().UTC(),
		RequestedTags: []string{"data"},
		Servers: map[string]ServerResult{
			"snowflake": {Status: "healthy", Port: 9890},
			"airbyte":   {Status: "unhealthy", Error: "dep_failed"},
			"airflow":   {Status: "stopped"},
		},
		Ready:    false,
		Blocking: []string{"airbyte"},
	}

	if len(report.Blocking) != 1 || report.Blocking[0] != "airbyte" {
		t.Errorf("blocking should contain only airbyte, got %v", report.Blocking)
	}
	// stopped servers should NOT be in blocking.
	for _, b := range report.Blocking {
		if b == "airflow" {
			t.Error("stopped server should not be in blocking list")
		}
	}
}
