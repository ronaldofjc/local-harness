package sessions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreStartAppendGet(t *testing.T) {
	tmp := t.TempDir()
	store := NewStore(tmp)

	// Start
	sessionID, err := store.Start("PREVC", "spec-123")
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if sessionID == "" {
		t.Fatal("expected non-empty session id")
	}

	// Get header
	header, events, err := store.Get(sessionID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if header.SessionID != sessionID {
		t.Errorf("expected session id %s, got %s", sessionID, header.SessionID)
	}
	if header.Workflow != "PREVC" {
		t.Errorf("expected workflow PREVC, got %s", header.Workflow)
	}
	if header.ContractID != "spec-123" {
		t.Errorf("expected contract_id spec-123, got %s", header.ContractID)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}

	// Append
	event := SessionEvent{
		Type:    "tool_call",
		Payload: json.RawMessage(`{"tool":"sensor.run"}`),
	}
	if err := store.Append(sessionID, event); err != nil {
		t.Fatalf("append failed: %v", err)
	}

	// Get after append
	header, events, err = store.Get(sessionID)
	if err != nil {
		t.Fatalf("get after append failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != "tool_call" {
		t.Errorf("expected event type tool_call, got %s", events[0].Type)
	}

	// Append second event
	event2 := SessionEvent{
		Type:    "decision",
		Payload: json.RawMessage(`{"decision":"proceed"}`),
	}
	if err := store.Append(sessionID, event2); err != nil {
		t.Fatalf("append second failed: %v", err)
	}

	_, events, _ = store.Get(sessionID)
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}
}

func TestStoreAppendNotFound(t *testing.T) {
	tmp := t.TempDir()
	store := NewStore(tmp)

	err := store.Append("nonexistent", SessionEvent{Type: "tool_call"})
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestStoreGetNotFound(t *testing.T) {
	tmp := t.TempDir()
	store := NewStore(tmp)

	_, _, err := store.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestStoreFileCreated(t *testing.T) {
	tmp := t.TempDir()
	store := NewStore(tmp)

	sessionID, _ := store.Start("", "")
	path := filepath.Join(tmp, ".local", "sessions", sessionID+".jsonl")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("expected session file to exist at %s", path)
	}
}

func TestStoreExists(t *testing.T) {
	tmp := t.TempDir()
	store := NewStore(tmp)

	if store.Exists("nonexistent") {
		t.Fatal("expected Exists=false for nonexistent session")
	}

	sessionID, _ := store.Start("", "")
	if !store.Exists(sessionID) {
		t.Fatal("expected Exists=true for existing session")
	}
}

func TestStoreDeleteOldSessions(t *testing.T) {
	tmp := t.TempDir()
	store := NewStore(tmp)

	// Cria arquivo antigo.
	oldPath := filepath.Join(tmp, ".local", "sessions", "old.jsonl")
	os.MkdirAll(filepath.Dir(oldPath), 0755)
	f, _ := os.Create(oldPath)
	f.Close()
	oldTime := time.Now().Add(-8 * 24 * time.Hour)
	os.Chtimes(oldPath, oldTime, oldTime)

	// Cria arquivo recente.
	newPath := filepath.Join(tmp, ".local", "sessions", "new.jsonl")
	f2, _ := os.Create(newPath)
	f2.Close()

	removed, err := store.DeleteOldSessions(7 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("delete old sessions failed: %v", err)
	}
	if removed != 1 {
		t.Fatalf("expected 1 removed, got %d", removed)
	}

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatal("expected old file to be deleted")
	}
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Fatal("expected new file to still exist")
	}
}

func TestStoreListSessionIDs(t *testing.T) {
	tmp := t.TempDir()
	store := NewStore(tmp)

	ids, err := store.ListSessionIDs()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected 0 ids, got %d", len(ids))
	}

	s1, _ := store.Start("", "")
	s2, _ := store.Start("", "")

	ids, err = store.ListSessionIDs()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 ids, got %d", len(ids))
	}

	m := make(map[string]bool)
	for _, id := range ids {
		m[id] = true
	}
	if !m[s1] || !m[s2] {
		t.Fatal("expected both session ids in list")
	}
}
