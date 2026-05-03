package sensors

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/ronaldofjc/local-harness/internal/common"
	harnessfs "github.com/ronaldofjc/local-harness/internal/harness/fs"
)

// Sensor representa um sensor registrado.
type Sensor struct {
	ID          string            `yaml:"id"`
	Kind        common.SensorKind `yaml:"kind"`
	Regulation  common.Regulation `yaml:"regulation"`
	Command     string            `yaml:"command"`
	Adapter     string            `yaml:"adapter"`
	Description string            `yaml:"description"`
	Defaults    map[string]string `yaml:"defaults,omitempty"`
}

// Repository define a interface para acesso a sensors.
type Repository interface {
	List(kind string, regulation common.Regulation) ([]Sensor, error)
	Get(id string) (*Sensor, error)
	Register(s Sensor) error
}

// FileSystemRepository implementa Repository a partir do filesystem.
type FileSystemRepository struct {
	root    string
	mu      sync.RWMutex
	sensors map[string]*Sensor
}

// NewFileSystemRepository cria um novo repositorio de sensors.
func NewFileSystemRepository(root string) *FileSystemRepository {
	return &FileSystemRepository{
		root:    root,
		sensors: make(map[string]*Sensor),
	}
}

// Load carrega todos os sensors do diretorio sensors/.
func (r *FileSystemRepository) Load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	clear(r.sensors)
	sensorsDir := filepath.Join(r.root, "sensors")

	entries, err := os.ReadDir(sensorsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read sensors dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(sensorsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read sensor %s: %w", path, err)
		}

		var s Sensor
		if err := yaml.Unmarshal(data, &s); err != nil {
			return fmt.Errorf("parse sensor %s: %w", path, err)
		}
		r.sensors[s.ID] = &s
	}
	return nil
}

// List lista sensors com filtros opcionais.
func (r *FileSystemRepository) List(kind string, regulation common.Regulation) ([]Sensor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Sensor
	for _, s := range r.sensors {
		if kind != "" && string(s.Kind) != kind {
			continue
		}
		if regulation != "" && s.Regulation != regulation {
			continue
		}
		result = append(result, *s)
	}
	return result, nil
}

// Get retorna um sensor pelo ID.
func (r *FileSystemRepository) Get(id string) (*Sensor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, ok := r.sensors[id]
	if !ok {
		return nil, fmt.Errorf("sensor not found: %s", id)
	}
	return s, nil
}

// Register adiciona ou atualiza um sensor no filesystem (idempotente).
func (r *FileSystemRepository) Register(s Sensor) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	sensorsDir := filepath.Join(r.root, "sensors")
	if err := os.MkdirAll(sensorsDir, 0755); err != nil {
		return fmt.Errorf("mkdir sensors: %w", err)
	}

	path := filepath.Join(sensorsDir, fmt.Sprintf("%s.yaml", s.ID))

	// Verifica se ja existe e e identico
	if existing, err := os.ReadFile(path); err == nil {
		var existingSensor Sensor
		if err := yaml.Unmarshal(existing, &existingSensor); err == nil {
			if sensorsEqual(existingSensor, s) {
				return nil // idempotente
			}
		}
	}

	data, err := yaml.Marshal(&s)
	if err != nil {
		return fmt.Errorf("marshal sensor: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write sensor: %w", err)
	}

	r.sensors[s.ID] = &s
	return nil
}

// StartWatcher inicia o watcher para recarregar sensors automaticamente.
func (r *FileSystemRepository) StartWatcher() error {
	w, err := harnessfs.NewWatcher(r.root)
	if err != nil {
		return err
	}
	w.OnEvent(func(ev harnessfs.Event) {
		if strings.Contains(ev.Path, "/sensors/") && strings.HasSuffix(ev.Path, ".yaml") {
			_ = r.Load()
		}
	})
	return nil
}

func sensorsEqual(a, b Sensor) bool {
	return a.ID == b.ID && a.Kind == b.Kind && a.Regulation == b.Regulation &&
		a.Command == b.Command && a.Adapter == b.Adapter
}
