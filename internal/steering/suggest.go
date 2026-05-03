package steering

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ronaldofjc/local-harness/internal/common"
)

// SuggestInput define os parametros de entrada para harness.steer.suggest.
type SuggestInput struct {
	WindowDays int `json:"windowDays,omitempty"`
}

// SuggestOutput define a resposta de harness.steer.suggest.
type SuggestOutput struct {
	Suggestions        []string `json:"suggestions"`
	WindowDays         int      `json:"windowDays"`
	EventCountAnalyzed int      `json:"eventCountAnalyzed"`
}

// Suggester gera sugestoes a partir do steering log.
type Suggester struct {
	log *Log
}

// NewSuggester cria um novo suggester.
func NewSuggester(log *Log) *Suggester {
	return &Suggester{log: log}
}

// Suggest analisa o steering log e retorna sugestoes de guides.
func (s *Suggester) Suggest(input SuggestInput) (*SuggestOutput, error) {
	windowDays := max(input.WindowDays, 7)

	events, err := s.log.ReadAll()
	if err != nil {
		return nil, err
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -windowDays)
	var filtered []Event
	for _, ev := range events {
		if ev.Timestamp.After(cutoff) || ev.Timestamp.Equal(cutoff) {
			filtered = append(filtered, ev)
		}
	}

	// Agrega violations por tipo e frequencia
	violationCounts := make(map[string]int)
	regulationCounts := make(map[common.Regulation]int)
	var totalViolations int

	for _, ev := range filtered {
		if ev.Passed {
			continue
		}
		for _, v := range ev.Violations {
			totalViolations++
			key := normalizeKey(v.What)
			violationCounts[key]++
			regulationCounts[ev.Regulation]++
		}
	}

	var suggestions []string

	// Sugere guides para violations recorrentes (> 2 ocorrencias)
	type pair struct {
		key   string
		count int
	}
	var pairs []pair
	for k, c := range violationCounts {
		pairs = append(pairs, pair{key: k, count: c})
	}
	slices.SortFunc(pairs, func(a, b pair) int {
		return cmp.Compare(b.count, a.count)
	})

	for _, p := range pairs {
		if p.count >= 2 {
			suggestions = append(suggestions, fmt.Sprintf(
				"Consider adding a guide for '%s' (seen %d times in %d days). Example: create .harness/guides/rules/%s.md",
				p.key, p.count, windowDays, sanitizeFilename(p.key),
			))
		}
	}

	// Sugere atencao a regulations com muitos problemas
	if regulationCounts[common.RegulationMaintainability] >= 3 {
		suggestions = append(suggestions, fmt.Sprintf(
			"Maintainability issues are frequent (%d violations). Consider running 'gofmt' and 'staticcheck' before commits.",
			regulationCounts[common.RegulationMaintainability],
		))
	}
	if regulationCounts[common.RegulationFitness] >= 3 {
		suggestions = append(suggestions, fmt.Sprintf(
			"Fitness issues are frequent (%d violations). Consider reviewing tests and type coverage.",
			regulationCounts[common.RegulationFitness],
		))
	}
	if regulationCounts[common.RegulationBehaviour] >= 3 {
		suggestions = append(suggestions, fmt.Sprintf(
			"Behaviour issues are frequent (%d violations). Consider adding contract specs for edge cases.",
			regulationCounts[common.RegulationBehaviour],
		))
	}

	// Sugestao generica se nao houver dados suficientes
	if len(filtered) < 5 {
		suggestions = append(suggestions, fmt.Sprintf(
			"Not enough events in the last %d days (%d events). Keep using sensors and judges to build the steering log.",
			windowDays, len(filtered),
		))
	}

	return &SuggestOutput{
		Suggestions:        suggestions,
		WindowDays:         windowDays,
		EventCountAnalyzed: len(filtered),
	}, nil
}

func normalizeKey(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, ".", "-")
	return s
}

func sanitizeFilename(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	return b.String()
}
