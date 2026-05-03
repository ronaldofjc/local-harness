package contracts

import (
	"fmt"
	"time"

	"github.com/ronaldofjc/local-harness/internal/common"
	"github.com/ronaldofjc/local-harness/internal/sensors"
)

// SensorRunner define a interface para executar sensores (usada por contract.spec.validate).
type SensorRunner interface {
	Run(id string, target string) (common.SensorOutput, error)
}

// Service orquestra operacoes de contracts.
type Service struct {
	specRepo   SpecRepository
	taskRepo   TaskRepository
	sensorSvc  *sensors.Service
}

// NewService cria um novo service de contracts.
func NewService(specRepo SpecRepository, taskRepo TaskRepository, sensorSvc *sensors.Service) *Service {
	return &Service{
		specRepo:  specRepo,
		taskRepo:  taskRepo,
		sensorSvc: sensorSvc,
	}
}

// ValidateInput define os parametros de entrada para spec.validate.
type ValidateInput struct {
	ID      string `json:"id"`
	Artifact string `json:"artifact"`
}

// ValidateCheckResult representa o resultado de um check individual.
type ValidateCheckResult struct {
	Kind    string              `json:"kind"`
	ID      string              `json:"id"`
	Passed  bool                `json:"passed"`
	Output  common.SensorOutput `json:"output,omitempty"`
	Pending *JudgePending       `json:"pending,omitempty"`
}

// JudgePending indica que um judge check requer review do agente.
type JudgePending struct {
	Instructions string                 `json:"instructions"`
	Schema       map[string]interface{} `json:"schema"`
	Context      map[string]interface{} `json:"context"`
	Submission   map[string]interface{} `json:"submission"`
}

// ValidateOutput define a resposta de contract.spec.validate.
type ValidateOutput struct {
	Tool         string                `json:"tool"`
	Regulation   common.Regulation     `json:"regulation"`
	Passed       bool                  `json:"passed"`
	Summary      string                `json:"summary"`
	Inconclusive bool                  `json:"inconclusive"`
	Violations   []common.Violation    `json:"violations"`
	Pending      []ValidateCheckResult `json:"pending,omitempty"`
}

// Validate executa os checks de uma spec contra um artefato.
func (s *Service) Validate(input ValidateInput) (*ValidateOutput, error) {
	spec, err := s.specRepo.Get(input.ID)
	if err != nil {
		return nil, err
	}

	output := &ValidateOutput{
		Tool:       fmt.Sprintf("contract.spec.validate:%s", spec.ID),
		Regulation: common.RegulationBehaviour,
		Passed:     true,
		Violations: []common.Violation{},
		Pending:    []ValidateCheckResult{},
	}

	var checkCount, passCount int

	for _, check := range spec.Checks {
		checkCount++
		switch check.Kind {
		case "sensor":
			result := s.runSensorCheck(check, input.Artifact)
			if result.Inconclusive {
				output.Inconclusive = true
			} else if !result.Passed {
				output.Passed = false
			}
			output.Violations = append(output.Violations, result.Violations...)
			if result.Passed {
				passCount++
			}

		case "judge":
			// Judge checks nao sao executados pelo servidor
			output.Passed = false
			output.Pending = append(output.Pending, ValidateCheckResult{
				Kind:   "judge",
				ID:     check.Rubric,
				Passed: false,
				Pending: &JudgePending{
					Instructions: fmt.Sprintf("Run judge.review with rubric_id=%s and target=%s", check.Rubric, input.Artifact),
				},
			})
		}
	}

	if len(output.Pending) > 0 {
		output.Summary = fmt.Sprintf("%d/%d checks passed, %d pending judge review", passCount, checkCount, len(output.Pending))
	} else if output.Inconclusive {
		output.Summary = fmt.Sprintf("%d/%d checks passed, some inconclusive", passCount, checkCount)
	} else {
		output.Summary = fmt.Sprintf("%d/%d checks passed", passCount, checkCount)
	}

	return output, nil
}

func (s *Service) runSensorCheck(check Check, artifact string) common.SensorOutput {
	if s.sensorSvc == nil {
		return common.SensorOutput{
			Tool:               check.ID,
			Passed:             false,
			Inconclusive:       true,
			InconclusiveReason: "sensor service not available",
		}
	}

	result, err := s.sensorSvc.Run(check.ID, artifact)
	if err != nil {
		return common.SensorOutput{
			Tool:               check.ID,
			Passed:             false,
			Inconclusive:       true,
			InconclusiveReason: err.Error(),
		}
	}
	return result
}

// TaskNextInput define os parametros de entrada para task.next.
type TaskNextInput struct {
	SpecID string `json:"spec_id"`
}

// TaskNextOutput define a resposta de task.next.
type TaskNextOutput struct {
	Task *Task `json:"task"`
}

// TaskNext retorna a proxima task pendente da spec.
func (s *Service) TaskNext(input TaskNextInput) (*TaskNextOutput, error) {
	spec, err := s.specRepo.Get(input.SpecID)
	if err != nil {
		return nil, err
	}

	// Carrega tasks existentes
	existingTasks, err := s.taskRepo.ListBySpec(input.SpecID)
	if err != nil {
		return nil, err
	}

	existingMap := make(map[string]*Task)
	for i := range existingTasks {
		existingMap[existingTasks[i].ID] = &existingTasks[i]
	}

	// Itera tasks da spec em ordem
	for _, specTask := range spec.Tasks {
		task, exists := existingMap[specTask.ID]
		if !exists {
			// Cria nova task pending
			newTask := &Task{
				ID:          specTask.ID,
				SpecID:      input.SpecID,
				Description: specTask.Description,
				Status:      TaskStatusInProgress,
				Evidence:    []Evidence{},
			}
			if err := s.taskRepo.Save(newTask); err != nil {
				return nil, err
			}
			return &TaskNextOutput{Task: newTask}, nil
		}

		if task.Status == TaskStatusPending {
			task.Status = TaskStatusInProgress
			if err := s.taskRepo.Save(task); err != nil {
				return nil, err
			}
			return &TaskNextOutput{Task: task}, nil
		}

		if task.Status == TaskStatusInProgress {
			return &TaskNextOutput{Task: task}, nil
		}
	}

	// Todas as tasks completas
	return &TaskNextOutput{Task: nil}, nil
}

// TaskCompleteInput define os parametros de entrada para task.complete.
type TaskCompleteInput struct {
	TaskID   string     `json:"task_id"`
	Evidence []Evidence `json:"evidence"`
}

// TaskCompleteOutput define a resposta de task.complete.
type TaskCompleteOutput struct {
	Task *Task `json:"task"`
}

// TaskComplete marca uma task como completed.
func (s *Service) TaskComplete(input TaskCompleteInput) (*TaskCompleteOutput, error) {
	task, err := s.taskRepo.Get(input.TaskID)
	if err != nil {
		return nil, err
	}

	if task.Status == TaskStatusCompleted {
		return nil, common.ErrTaskAlreadyCompleted
	}

	// Anexa evidencias (append-only)
	now := time.Now().UTC()
	for i := range input.Evidence {
		if input.Evidence[i].Timestamp.IsZero() {
			input.Evidence[i].Timestamp = now
		}
		task.Evidence = append(task.Evidence, input.Evidence[i])
	}

	task.Status = TaskStatusCompleted
	task.CompletedAt = &now

	if err := s.taskRepo.Save(task); err != nil {
		return nil, err
	}

	return &TaskCompleteOutput{Task: task}, nil
}
