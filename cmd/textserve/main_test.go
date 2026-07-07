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
	if len(nonEmptyLines(out)) == 0 {
		t.Error("list: expected at least one server")
	}
}

func TestList_TagDocker(t *testing.T) {
	out, err := captureList("--tag", "docker")
	if err != nil {
		t.Fatalf("list --tag docker: %v", err)
	}
	// airflow and sentry must not appear
	for _, line := range nonEmptyLines(out) {
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
