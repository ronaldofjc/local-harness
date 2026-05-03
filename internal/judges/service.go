package judges

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ronaldofjc/local-harness/internal/common"
)

// Service orquestra operacoes de judges.
type Service struct {
	repo      Repository
	validator *Validator
}

// NewService cria um novo service de judges.
func NewService(repo Repository, validator *Validator) *Service {
	return &Service{repo: repo, validator: validator}
}

// List lista rubrics disponiveis.
func (s *Service) List(filterID string) ([]Rubric, error) {
	return s.repo.List(filterID)
}

// ReviewInput define os parametros de entrada para review.
type ReviewInput struct {
	RubricID string `json:"rubric_id"`
	Target   string `json:"target"`
	SpecID   string `json:"spec_id,omitempty"`
}

// ReviewOutput define a resposta da fase 1 (preparacao).
type ReviewOutput struct {
	RubricID     string                 `json:"rubric_id"`
	Regulation   common.Regulation      `json:"regulation"`
	Instructions string                 `json:"instructions"`
	Schema       map[string]any `json:"schema"`
	Context      ReviewContext          `json:"context"`
	Submission   ReviewSubmission       `json:"submission"`
}

// ReviewContext fornece contexto carregado pelo servidor.
type ReviewContext struct {
	Spec   string `json:"spec,omitempty"`
	Target string `json:"target"`
}

// ReviewSubmission indica qual tool chamar a seguir.
type ReviewSubmission struct {
	Tool        string                 `json:"tool"`
	Correlation map[string]any `json:"correlation"`
}

// Review renderiza o prompt da rubric para o cliente MCP avaliar.
func (s *Service) Review(input ReviewInput) (*ReviewOutput, error) {
	rubric, err := s.repo.Get(input.RubricID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", common.ErrRubricNotFound, err)
	}

	// Renderiza prompt com placeholders
	instructions := rubric.Prompt
	instructions = strings.ReplaceAll(instructions, "<<<TARGET>>>", input.Target)

	var specContent string
	if input.SpecID != "" {
		// TODO: carregar spec do contracts repo quando implementado
		specContent = fmt.Sprintf("id: %s\n(spec content not yet loaded)", input.SpecID)
		instructions = strings.ReplaceAll(instructions, "<<<SPEC>>>", specContent)
	} else {
		instructions = strings.ReplaceAll(instructions, "<<<SPEC>>>", "(no spec provided)")
	}

	instructions = strings.ReplaceAll(instructions, "<<<SCHEMA>>>", rubric.Schema)

	// Parse schema como map
	var schemaMap map[string]any
	_ = json.Unmarshal([]byte(rubric.Schema), &schemaMap)

	return &ReviewOutput{
		RubricID:     rubric.ID,
		Regulation:   rubric.Regulation,
		Instructions: instructions,
		Schema:       schemaMap,
		Context: ReviewContext{
			Spec:   specContent,
			Target: input.Target,
		},
		Submission: ReviewSubmission{
			Tool: "judge.record",
			Correlation: map[string]any{
				"rubric_id": rubric.ID,
				"target":    input.Target,
			},
		},
	}, nil
}

// RecordInput define os parametros de entrada para record.
type RecordInput struct {
	RubricID string                 `json:"rubric_id"`
	Target   string                 `json:"target"`
	SpecID   string                 `json:"spec_id,omitempty"`
	Result   map[string]any `json:"result"`
}

// Record processa o verdict do cliente, valida pelo schema e retorna envelope normalizado.
func (s *Service) Record(input RecordInput) (common.SensorOutput, error) {
	rubric, err := s.repo.Get(input.RubricID)
	if err != nil {
		return common.SensorOutput{}, fmt.Errorf("%w: %v", common.ErrRubricNotFound, err)
	}

	// Verifica se o cliente sinalizou inconclusivo
	if inconclusive, ok := input.Result["inconclusive"].(bool); ok && inconclusive {
		reason, _ := input.Result["inconclusiveReason"].(string)
		return common.SensorOutput{
			Tool:               rubric.ID,
			Regulation:         rubric.Regulation,
			Passed:             false,
			Summary:            fmt.Sprintf("%s: inconclusive", rubric.ID),
			Inconclusive:       true,
			InconclusiveReason: reason,
			Violations:         []common.Violation{},
		}, nil
	}

	// Valida pelo schema
	resultJSON, _ := json.Marshal(input.Result)
	if err := s.validator.Validate(string(resultJSON), rubric.Schema); err != nil {
		return common.SensorOutput{
			Tool:               rubric.ID,
			Regulation:         rubric.Regulation,
			Passed:             false,
			Summary:            fmt.Sprintf("%s: result failed schema validation", rubric.ID),
			Inconclusive:       true,
			InconclusiveReason: fmt.Sprintf("result_failed_schema_validation: %v", err),
			Violations:         []common.Violation{},
		}, nil
	}

	// Extrai campos do resultado
	passed, _ := input.Result["passed"].(bool)
	summary, _ := input.Result["summary"].(string)
	if summary == "" {
		summary = fmt.Sprintf("%s: review completed", rubric.ID)
	}

	// Converte violations
	var violations []common.Violation
	if vArray, ok := input.Result["violations"].([]any); ok {
		for _, v := range vArray {
			if vMap, ok := v.(map[string]any); ok {
				violations = append(violations, mapToViolation(vMap))
			}
		}
	}

	return common.SensorOutput{
		Tool:       rubric.ID,
		Regulation: rubric.Regulation,
		Passed:     passed,
		Summary:    summary,
		Violations: violations,
	}, nil
}

func mapToViolation(m map[string]any) common.Violation {
	v := common.Violation{}
	if s, ok := m["severity"].(string); ok {
		v.Severity = common.Severity(s)
	}
	if s, ok := m["what"].(string); ok {
		v.What = s
	}
	if s, ok := m["why"].(string); ok {
		v.Why = s
	}
	if s, ok := m["remediation"].(string); ok {
		v.Remediation = s
	}
	if arr, ok := m["filesAffected"].([]any); ok {
		for _, f := range arr {
			if s, ok := f.(string); ok {
				v.FilesAffected = append(v.FilesAffected, s)
			}
		}
	}
	if arr, ok := m["linesAffected"].([]any); ok {
		for _, line := range arr {
			if tuple, ok := line.([]any); ok && len(tuple) == 2 {
				start, _ := toInt(tuple[0])
				end, _ := toInt(tuple[1])
				v.LinesAffected = append(v.LinesAffected, [2]int{start, end})
			}
		}
	}
	if s, ok := m["guideUri"].(string); ok {
		v.GuideURI = s
	}
	return v
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}
