package main

import (
	"bytes"
	"strings"
	"testing"
)

func captureList(args ...string) (string, error) {
	root := buildRoot()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetArgs(append([]string{"list"}, args...))
	err := root.Execute()
	return buf.String(), err
}

func TestList_AllServers(t *testing.T) {
	out, err := captureList()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	lines := nonEmptyLines(out)
	if len(lines) != 11 {
		t.Errorf("list: got %d servers, want 11\n%s", len(lines), out)
	}
}

func TestList_TagDocker(t *testing.T) {
	out, err := captureList("--tag", "docker")
	if err != nil {
		t.Fatalf("list --tag docker: %v", err)
	}
	lines := nonEmptyLines(out)
	if len(lines) != 9 {
		t.Errorf("list --tag docker: got %d servers, want 9\n%s", len(lines), out)
	}
	// airflow and sentry must not appear
	for _, line := range lines {
		if line == "airflow" || line == "sentry" {
			t.Errorf("list --tag docker: unexpected server %q", line)
		}
	}
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			out = append(out, strings.TrimSpace(l))
		}
	}
	return out
}
