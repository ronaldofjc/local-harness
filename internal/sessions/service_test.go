package sessions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestServiceStartReusesActiveSession(t *testing.T) {
	tmp := t.TempDir()
	store := NewStore(tmp)
	svc := NewService(store)

	// Primeira chamada cria nova sessão.
	out1, err := svc.Start(StartInput{Workflow: "PREVC", ContractID: "spec-1"})
	if err != nil {
		t.Fatalf("start 1 failed: %v", err)
	}
	if out1.SessionID == "" {
		t.Fatal("expected non-empty session id")
	}
	if out1.Reused {
		t.Fatal("expected reused=false on first start")
	}
	if out1.StartedAt == "" {
		t.Fatal("expected startedAt to be set")
	}

	// Segunda chamada reutiliza a mesma sessão.
	out2, err := svc.Start(StartInput{Workflow: "OTHER"})
	if err != nil {
		t.Fatalf("start 2 failed: %v", err)
	}
	if out2.SessionID != out1.SessionID {
		t.Fatalf("expected same session id, got %s vs %s", out1.SessionID, out2.SessionID)
	}
	if !out2.Reused {
		t.Fatal("expected reused=true on second start")
	}
}

func TestServiceStartForceNew(t *testing.T) {
	tmp := t.TempDir()
	store := NewStore(tmp)
	svc := NewService(store)

	out1, _ := svc.Start(StartInput{Workflow: "PREVC"})
	out2, err := svc.Start(StartInput{Workflow: "OTHER", ForceNew: true})
	if err != nil {
		t.Fatalf("force_new start failed: %v", err)
	}
	if out2.SessionID == "" {
		t.Fatal("expected non-empty session id")
	}
	if out2.Reused {
		t.Fatal("expected reused=false on force_new")
	}
	if out2.SessionID == out1.SessionID {
		t.Fatal("expected different session id on force_new")
	}

	// Sem force_new, agora deve reusar a nova ativa.
	out3, _ := svc.Start(StartInput{})
	if out3.SessionID != out2.SessionID {
		t.Fatalf("expected reuse of force_new session, got %s vs %s", out2.SessionID, out3.SessionID)
	}
	if !out3.Reused {
		t.Fatal("expected reused=true")
	}
}

func TestServiceStartRecreatesIfFileDeleted(t *testing.T) {
	tmp := t.TempDir()
	store := NewStore(tmp)
	svc := NewService(store)

	out1, _ := svc.Start(StartInput{Workflow: "PREVC"})
	path := filepath.Join(tmp, ".local", "sessions", out1.SessionID+".jsonl")

	// Deleta arquivo externamente.
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove session file: %v", err)
	}

	out2, err := svc.Start(StartInput{})
	if err != nil {
		t.Fatalf("start after deletion failed: %v", err)
	}
	if out2.SessionID == out1.SessionID {
		t.Fatal("expected new session id after file deletion")
	}
	if out2.Reused {
		t.Fatal("expected reused=false after file deletion")
	}
}

func TestServiceHousekeeping(t *testing.T) {
	tmp := t.TempDir()
	store := NewStore(tmp)
	svc := NewService(store)

	// Cria uma sessão antiga manipulando o arquivo diretamente.
	oldSessionID := "oldsession123456"
	oldPath := filepath.Join(tmp, ".local", "sessions", oldSessionID+".jsonl")
	if err := os.MkdirAll(filepath.Dir(oldPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	f, err := os.Create(oldPath)
	if err != nil {
		t.Fatalf("create old file: %v", err)
	}
	f.Close()

	// Altera o tempo de modificação para 8 dias atrás.
	oldTime := time.Now().Add(-8 * 24 * time.Hour)
	if err := os.Chtimes(oldPath, oldTime, oldTime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	// Nova sessão deve disparar housekeeping e remover a antiga.
	out, err := svc.Start(StartInput{Workflow: "PREVC"})
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if out.SessionID == oldSessionID {
		t.Fatal("expected new session, not old one")
	}

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatal("expected old session file to be removed by housekeeping")
	}
}

func TestServiceAppendAndGet(t *testing.T) {
	tmp := t.TempDir()
	store := NewStore(tmp)
	svc := NewService(store)

	out, _ := svc.Start(StartInput{})

	event := SessionEvent{
		Type:    "tool_call",
		Payload: json.RawMessage(`{"tool":"sensor.run"}`),
	}
	appendOut, err := svc.Append(AppendInput{SessionID: out.SessionID, Event: mustMarshal(t, event)})
	if err != nil {
		t.Fatalf("append failed: %v", err)
	}
	if !appendOut.Appended {
		t.Fatal("expected appended=true")
	}
	if appendOut.EventCount != 1 {
		t.Fatalf("expected eventCount=1, got %d", appendOut.EventCount)
	}

	getOut, err := svc.Get(GetInput{SessionID: out.SessionID})
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if len(getOut.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(getOut.Events))
	}
}

func mustMarshal(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return json.RawMessage(b)
}
