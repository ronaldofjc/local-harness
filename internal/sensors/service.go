package sensors

import (
	"fmt"

	"github.com/ronaldofjc/local-harness/internal/common"
)

// Service orquestra a execucao de sensors.
type Service struct {
	repo   Repository
	runner *Runner
}

// NewService cria um novo service de sensors.
func NewService(repo Repository, runner *Runner) *Service {
	return &Service{repo: repo, runner: runner}
}

// List lista sensors com filtros.
func (s *Service) List(kind string, regulation common.Regulation) ([]Sensor, error) {
	return s.repo.List(kind, regulation)
}

// Run executa um sensor pelo ID.
func (s *Service) Run(id string, target string) (common.SensorOutput, error) {
	sensor, err := s.repo.Get(id)
	if err != nil {
		return common.SensorOutput{}, fmt.Errorf("%w: %v", common.ErrSensorNotFound, err)
	}
	return s.runner.Run(sensor, target), nil
}

// Register registra um novo sensor.
func (s *Service) Register(sensor Sensor) error {
	return s.repo.Register(sensor)
}
