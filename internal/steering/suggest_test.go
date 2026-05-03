package steering

import (
	"testing"
	"time"

	"github.com/ronaldofjc/local-harness/internal/common"
)

func TestSuggestEmptyLog(t *testing.T) {
	tmp := t.TempDir()
	log := NewLog(tmp)
	suggester := NewSuggester(log)

	output, err := suggester.Suggest(SuggestInput{WindowDays: 7})
	if err != nil {
		t.Fatalf("suggest failed: %v", err)
	}

	if output.WindowDays != 7 {
		t.Errorf("expected windowDays 7, got %d", output.WindowDays)
	}
	if output.EventCountAnalyzed != 0 {
		t.Errorf("expected 0 events analyzed, got %d", output.EventCountAnalyzed)
	}
	if len(output.Suggestions) == 0 {
		t.Fatal("expected at least one suggestion for empty log")
	}
	// Deve conter mensagem sobre dados insuficientes
	found := false
	for _, s := range output.Suggestions {
		if contains(s, "Not enough events") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected suggestion about not enough events, got %v", output.Suggestions)
	}
}

func TestSuggestWithViolations(t *testing.T) {
	tmp := t.TempDir()
	log := NewLog(tmp)
	suggester := NewSuggester(log)

	now := time.Now().UTC()
	// Adiciona 3 eventos com a mesma violation
	for range 3 {
		_ = log.Append(Event{
			Timestamp:  now,
			Source:     "sensor",
			Tool:       "staticcheck",
			Regulation: common.RegulationMaintainability,
			Passed:     false,
			Violations: []common.Violation{
				{What: "unused variable", Severity: common.SeverityWarning},
			},
		})
	}

	output, err := suggester.Suggest(SuggestInput{WindowDays: 7})
	if err != nil {
		t.Fatalf("suggest failed: %v", err)
	}

	if output.EventCountAnalyzed != 3 {
		t.Errorf("expected 3 events analyzed, got %d", output.EventCountAnalyzed)
	}

	// Deve sugerir guide para "unused-variable"
	found := false
	for _, s := range output.Suggestions {
		if contains(s, "unused-variable") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected suggestion about unused-variable, got %v", output.Suggestions)
	}
}

func TestSuggestRegulationThreshold(t *testing.T) {
	tmp := t.TempDir()
	log := NewLog(tmp)
	suggester := NewSuggester(log)

	now := time.Now().UTC()
	// 4 maintainability violations
	for range 4 {
		_ = log.Append(Event{
			Timestamp:  now,
			Source:     "sensor",
			Tool:       "gofmt",
			Regulation: common.RegulationMaintainability,
			Passed:     false,
			Violations: []common.Violation{
				{What: "formatting issue", Severity: common.SeverityWarning},
			},
		})
	}

	output, err := suggester.Suggest(SuggestInput{WindowDays: 7})
	if err != nil {
		t.Fatalf("suggest failed: %v", err)
	}

	found := false
	for _, s := range output.Suggestions {
		if contains(s, "Maintainability issues are frequent") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected suggestion about maintainability frequency, got %v", output.Suggestions)
	}
}

func TestSuggestWindowFiltering(t *testing.T) {
	tmp := t.TempDir()
	log := NewLog(tmp)
	suggester := NewSuggester(log)

	// Evento antigo (15 dias atras)
	_ = log.Append(Event{
		Timestamp:  time.Now().UTC().AddDate(0, 0, -15),
		Source:     "sensor",
		Tool:       "staticcheck",
		Regulation: common.RegulationMaintainability,
		Passed:     false,
		Violations: []common.Violation{{What: "old issue"}},
	})

	// Evento recente
	_ = log.Append(Event{
		Timestamp:  time.Now().UTC(),
		Source:     "sensor",
		Tool:       "staticcheck",
		Regulation: common.RegulationMaintainability,
		Passed:     false,
		Violations: []common.Violation{{What: "new issue"}},
	})

	output, err := suggester.Suggest(SuggestInput{WindowDays: 7})
	if err != nil {
		t.Fatalf("suggest failed: %v", err)
	}

	if output.EventCountAnalyzed != 1 {
		t.Errorf("expected 1 event in 7-day window, got %d", output.EventCountAnalyzed)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
