package workflows

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileSystemRepositoryLoad(t *testing.T) {
	tmp := t.TempDir()
	wfDir := filepath.Join(tmp, "workflows")
	if err := os.MkdirAll(wfDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Cria workflow de teste
	content := "# Test Workflow\n\nThis is a test workflow.\n"
	if err := os.WriteFile(filepath.Join(wfDir, "test-wf.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	repo := NewFileSystemRepository(tmp)
	if err := repo.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	wf, err := repo.Get("test-wf")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if wf.ID != "test-wf" {
		t.Errorf("expected id test-wf, got %s", wf.ID)
	}
	if wf.Name != "Test Workflow" {
		t.Errorf("expected name 'Test Workflow', got %s", wf.Name)
	}
	if wf.Content != content {
		t.Errorf("content mismatch")
	}
}

func TestFileSystemRepositoryAll(t *testing.T) {
	tmp := t.TempDir()
	wfDir := filepath.Join(tmp, "workflows")
	if err := os.MkdirAll(wfDir, 0755); err != nil {
		t.Fatal(err)
	}

	_ = os.WriteFile(filepath.Join(wfDir, "a.md"), []byte("# A\n"), 0644)
	_ = os.WriteFile(filepath.Join(wfDir, "b.md"), []byte("# B\n"), 0644)

	repo := NewFileSystemRepository(tmp)
	if err := repo.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	all, err := repo.All()
	if err != nil {
		t.Fatalf("all failed: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 workflows, got %d", len(all))
	}
}

func TestFileSystemRepositoryGetNotFound(t *testing.T) {
	tmp := t.TempDir()
	repo := NewFileSystemRepository(tmp)
	_ = repo.Load()

	_, err := repo.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent workflow")
	}
}

func TestToPrompts(t *testing.T) {
	wfs := []Workflow{
		{ID: "PREVC", Description: "Plan workflow"},
		{ID: "bug-fix", Description: "Fix workflow"},
	}
	prompts := ToPrompts(wfs)
	if len(prompts) != 2 {
		t.Errorf("expected 2 prompts, got %d", len(prompts))
	}
	if prompts[0].Name != "workflow.PREVC" {
		t.Errorf("expected workflow.PREVC, got %s", prompts[0].Name)
	}
}

func TestGetPromptMessages(t *testing.T) {
	wf := &Workflow{Content: "Hello workflow"}
	msgs := GetPromptMessages(wf)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Role != "user" {
		t.Errorf("expected role user, got %s", msgs[0].Role)
	}
	if msgs[0].Content != "Hello workflow" {
		t.Errorf("content mismatch")
	}
}
