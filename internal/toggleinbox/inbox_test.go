package toggleinbox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paperworlds/textserve/internal/togglestate"
)

func setEnv(t *testing.T) (stateDir, logPath string) {
	t.Helper()
	stateDir = t.TempDir()
	logDir := t.TempDir()
	logPath = filepath.Join(logDir, "toggle.log")
	t.Setenv("TEXTSERVE_STATE_DIR", stateDir)
	t.Setenv("TEXTSERVE_LOG_PATH", logPath)
	return
}

func TestRequest_CreatesPendingEntryAndLog(t *testing.T) {
	_, logPath := setEnv(t)

	e, err := Request("memory", ActionEnable, "test")
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	if e.Status != StatusPending {
		t.Errorf("status: got %q want %q", e.Status, StatusPending)
	}
	if e.ID == "" {
		t.Error("ID empty")
	}

	got, err := Get(e.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Bundle != "memory" || got.Action != ActionEnable {
		t.Errorf("got %+v", got)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(data), `"event":"request"`) {
		t.Errorf("expected request event in log, got %s", data)
	}
}

func TestRequest_InvalidAction(t *testing.T) {
	setEnv(t)
	if _, err := Request("memory", "frobnicate", "test"); err == nil {
		t.Fatal("expected error for invalid action")
	}
}

func TestApprove_AppliesOverlayAndMarksApproved(t *testing.T) {
	setEnv(t)

	e, err := Request("memory", ActionEnable, "test")
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	approved, err := Approve(e.ID)
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if approved.Status != StatusApproved {
		t.Errorf("status: got %q want %q", approved.Status, StatusApproved)
	}

	overlay, err := togglestate.Load()
	if err != nil {
		t.Fatalf("Load overlay: %v", err)
	}
	if v, ok := overlay.Lookup("memory"); !ok || !v {
		t.Errorf("overlay memory: got %v ok=%v, want true", v, ok)
	}
}

func TestApprove_Disable(t *testing.T) {
	setEnv(t)
	e, _ := Request("github", ActionDisable, "test")
	if _, err := Approve(e.ID); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	overlay, _ := togglestate.Load()
	if v, ok := overlay.Lookup("github"); !ok || v {
		t.Errorf("overlay github: got %v ok=%v, want false", v, ok)
	}
}

func TestDeny_NoOverlayChange(t *testing.T) {
	setEnv(t)
	e, _ := Request("memory", ActionEnable, "test")
	denied, err := Deny(e.ID, "not now")
	if err != nil {
		t.Fatalf("Deny: %v", err)
	}
	if denied.Status != StatusDenied {
		t.Errorf("status: got %q want %q", denied.Status, StatusDenied)
	}
	if denied.Reason != "not now" {
		t.Errorf("reason: got %q want %q", denied.Reason, "not now")
	}
	overlay, _ := togglestate.Load()
	if _, ok := overlay.Lookup("memory"); ok {
		t.Error("deny must not mutate overlay")
	}
}

func TestApprove_AlreadyResolved(t *testing.T) {
	setEnv(t)
	e, _ := Request("memory", ActionEnable, "test")
	if _, err := Approve(e.ID); err != nil {
		t.Fatalf("first Approve: %v", err)
	}
	if _, err := Approve(e.ID); err == nil {
		t.Fatal("expected error approving already-approved entry")
	}
}

func TestList_FilterByStatus(t *testing.T) {
	setEnv(t)

	a, _ := Request("a", ActionEnable, "t")
	b, _ := Request("b", ActionEnable, "t")
	_, _ = Request("c", ActionEnable, "t")
	if _, err := Approve(a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := Deny(b.ID, ""); err != nil {
		t.Fatal(err)
	}

	pending, err := List(StatusPending)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].Bundle != "c" {
		t.Errorf("pending: got %+v, want 1 entry for bundle c", pending)
	}

	all, err := List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Errorf("all: got %d entries, want 3", len(all))
	}
}
