// Package toggleinbox is the file-based queue for bundle toggle requests.
// A request lives at $TEXTSERVE_STATE_DIR/toggle-inbox/<id>.yaml; transitions
// (request, approve, deny) are appended as JSON lines to the toggle log.
package toggleinbox

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/paperworlds/textserve/internal/togglestate"
)

const (
	subdirInbox = "toggle-inbox"
	logRelPath  = ".local/log/textserve-toggle.log"

	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusDenied   = "denied"

	ActionEnable  = "enable"
	ActionDisable = "disable"
)

// Entry is a single toggle request and its lifecycle.
type Entry struct {
	ID         string    `yaml:"id"`
	Bundle     string    `yaml:"bundle"`
	Action     string    `yaml:"action"`
	Status     string    `yaml:"status"`
	Requester  string    `yaml:"requester,omitempty"`
	Reason     string    `yaml:"reason,omitempty"`
	CreatedAt  time.Time `yaml:"created_at"`
	ResolvedAt time.Time `yaml:"resolved_at,omitempty"`
}

// InboxDir returns the absolute path to the toggle-inbox directory.
func InboxDir() (string, error) {
	dir, err := togglestate.StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, subdirInbox), nil
}

// LogPath returns the absolute path to the toggle log file.
// Honors $TEXTSERVE_LOG_PATH; otherwise ~/.local/log/textserve-toggle.log.
func LogPath() (string, error) {
	if p := os.Getenv("TEXTSERVE_LOG_PATH"); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, logRelPath), nil
}

func randSuffix() string {
	var b [3]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// Request enqueues a new toggle request and returns the created entry.
// Caller supplies a requester label (e.g. "mcp", "cli") for the audit log.
func Request(bundle, action, requester string) (*Entry, error) {
	if action != ActionEnable && action != ActionDisable {
		return nil, fmt.Errorf("invalid action %q (want enable|disable)", action)
	}
	if strings.TrimSpace(bundle) == "" {
		return nil, fmt.Errorf("bundle name required")
	}
	now := time.Now().UTC()
	id := fmt.Sprintf("%s-%s-%s-%s",
		now.Format("20060102T150405Z"), bundle, action, randSuffix())

	e := &Entry{
		ID:        id,
		Bundle:    bundle,
		Action:    action,
		Status:    StatusPending,
		Requester: requester,
		CreatedAt: now,
	}
	if err := writeEntry(e); err != nil {
		return nil, err
	}
	if err := appendLog("request", e, ""); err != nil {
		return nil, err
	}
	return e, nil
}

// List returns all entries in the inbox, sorted by creation time (oldest first).
// Pass non-empty status to filter (e.g. StatusPending).
func List(status string) ([]Entry, error) {
	dir, err := InboxDir()
	if err != nil {
		return nil, err
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read inbox: %w", err)
	}
	var out []Entry
	for _, de := range ents {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".yaml") {
			continue
		}
		e, err := readEntry(strings.TrimSuffix(de.Name(), ".yaml"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "toggleinbox: skipping %s: %v\n", de.Name(), err)
			continue
		}
		if status != "" && e.Status != status {
			continue
		}
		out = append(out, *e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

// Get returns a single entry by id.
func Get(id string) (*Entry, error) { return readEntry(id) }

// Approve marks the entry approved and applies the toggle to the overlay state.
func Approve(id string) (*Entry, error) {
	e, err := readEntry(id)
	if err != nil {
		return nil, err
	}
	if e.Status != StatusPending {
		return nil, fmt.Errorf("entry %s already %s", id, e.Status)
	}

	overlay, err := togglestate.Load()
	if err != nil {
		return nil, err
	}
	overlay.Set(e.Bundle, e.Action == ActionEnable)
	if err := overlay.Save(); err != nil {
		return nil, err
	}

	e.Status = StatusApproved
	e.ResolvedAt = time.Now().UTC()
	if err := writeEntry(e); err != nil {
		return nil, err
	}
	if err := appendLog("approve", e, ""); err != nil {
		return nil, err
	}
	return e, nil
}

// Deny marks the entry denied without mutating overlay state.
func Deny(id, reason string) (*Entry, error) {
	e, err := readEntry(id)
	if err != nil {
		return nil, err
	}
	if e.Status != StatusPending {
		return nil, fmt.Errorf("entry %s already %s", id, e.Status)
	}
	e.Status = StatusDenied
	e.Reason = reason
	e.ResolvedAt = time.Now().UTC()
	if err := writeEntry(e); err != nil {
		return nil, err
	}
	if err := appendLog("deny", e, reason); err != nil {
		return nil, err
	}
	return e, nil
}

// EntryPath returns the absolute path where the entry for id is stored.
// The file may or may not exist on disk.
func EntryPath(id string) (string, error) { return entryPath(id) }

func entryPath(id string) (string, error) {
	dir, err := InboxDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, id+".yaml"), nil
}

func readEntry(id string) (*Entry, error) {
	path, err := entryPath(id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read entry %s: %w", id, err)
	}
	var e Entry
	if err := yaml.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("parse entry %s: %w", id, err)
	}
	return &e, nil
}

func writeEntry(e *Entry) error {
	path, err := entryPath(e.ID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir inbox: %w", err)
	}
	data, err := yaml.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal entry: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write entry: %w", err)
	}
	return os.Rename(tmp, path)
}

type logLine struct {
	TS        string `json:"ts"`
	Event     string `json:"event"`
	ID        string `json:"id"`
	Bundle    string `json:"bundle"`
	Action    string `json:"action"`
	Status    string `json:"status"`
	Requester string `json:"requester,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

func appendLog(event string, e *Entry, reason string) error {
	path, err := LogPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir log dir: %w", err)
	}
	line := logLine{
		TS:        time.Now().UTC().Format(time.RFC3339Nano),
		Event:     event,
		ID:        e.ID,
		Bundle:    e.Bundle,
		Action:    e.Action,
		Status:    e.Status,
		Requester: e.Requester,
		Reason:    reason,
	}
	data, err := json.Marshal(line)
	if err != nil {
		return fmt.Errorf("marshal log line: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write log: %w", err)
	}
	return nil
}
