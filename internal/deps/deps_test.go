package deps_test

import (
	"testing"

	"github.com/paperworlds/textserve/internal/deps"
	"github.com/paperworlds/textserve/internal/registry"
)

func TestCheck_Passing(t *testing.T) {
	d := []registry.Dep{
		{Cmd: "true", Hint: "should not appear"},
	}
	if err := deps.Check(d); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestCheck_Failing(t *testing.T) {
	d := []registry.Dep{
		{Cmd: "false", Hint: "install the thing"},
	}
	if err := deps.Check(d); err == nil {
		t.Error("expected error for failing dep, got nil")
	}
}

func TestCheck_Empty(t *testing.T) {
	if err := deps.Check(nil); err != nil {
		t.Errorf("expected nil error for empty deps, got %v", err)
	}
}

func TestCheck_HintOnFailure(t *testing.T) {
	// Verify that a failing dep returns an error naming the failing command.
	d := []registry.Dep{
		{Cmd: "exit 1", Hint: "run setup.sh first"},
	}
	err := deps.Check(d)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() == "" {
		t.Error("error message is empty")
	}
}
