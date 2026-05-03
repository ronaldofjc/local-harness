package contracts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/ronaldofjc/local-harness/internal/common"
)

// TaskStatus representa o status de uma task.
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
)

// Task representa uma task persistida.
type Task struct {
	ID          string      `yaml:"id" json:"id"`
	SpecID      string      `yaml:"spec_id" json:"spec_id"`
	Description string      `yaml:"description" json:"description"`
	Status      TaskStatus  `yaml:"status" json:"status"`
	Evidence    []Evidence  `yaml:"evidence" json:"evidence"`
	CompletedAt *time.Time  `yaml:"completedAt,omitempty" json:"completedAt,omitempty"`
}

// Evidence representa uma evidencia anexada a uma task.
type Evidence struct {
	Kind      string    `yaml:"kind" json:"kind"` // sensor_run | judge_review | pr_link | note
	Timestamp time.Time `yaml:"timestamp" json:"timestamp"`
	// Campos especificos por kind
	Sensor string `yaml:"sensor,omitempty" json:"sensor,omitempty"`
	Passed *bool  `yaml:"passed,omitempty" json:"passed,omitempty"`
	Rubric string `yaml:"rubric,omitempty" json:"rubric,omitempty"`
	URL    string `yaml:"url,omitempty" json:"url,omitempty"`
	Text   string `yaml:"text,omitempty" json:"text,omitempty"`
}

// TaskRepository define a interface para persistencia de tasks.
type TaskRepository interface {
	Get(id string) (*Task, error)
	Save(task *Task) error
	ListBySpec(specID string) ([]Task, error)
}

// FileSystemTaskRepository implementa TaskRepository com YAML em disco.
type FileSystemTaskRepository struct {
	root  string
	mu    sync.RWMutex
	tasks map[string]*Task
}

// NewFileSystemTaskRepository cria um novo repositorio de tasks.
func NewFileSystemTaskRepository(root string) *FileSystemTaskRepository {
	return &FileSystemTaskRepository{
		root:  root,
		tasks: make(map[string]*Task),
	}
}

// Load carrega todas as tasks do diretorio contracts/tasks/.
func (r *FileSystemTaskRepository) Load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	clear(r.tasks)
	tasksDir := filepath.Join(r.root, "contracts", "tasks")

	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read tasks dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(tasksDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read task %s: %w", path, err)
		}

		var task Task
		if err := yaml.Unmarshal(data, &task); err != nil {
			return fmt.Errorf("parse task %s: %w", path, err)
		}
		r.tasks[task.ID] = &task
	}
	return nil
}

// Get retorna uma task pelo ID.
func (r *FileSystemTaskRepository) Get(id string) (*Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, ok := r.tasks[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", common.ErrTaskNotFound, id)
	}
	return task, nil
}

// Save persiste uma task em disco.
func (r *FileSystemTaskRepository) Save(task *Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tasksDir := filepath.Join(r.root, "contracts", "tasks")
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		return fmt.Errorf("mkdir tasks: %w", err)
	}

	path := filepath.Join(tasksDir, fmt.Sprintf("%s.yaml", task.ID))
	data, err := yaml.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write task: %w", err)
	}

	r.tasks[task.ID] = task
	return nil
}

// ListBySpec lista todas as tasks de uma spec.
func (r *FileSystemTaskRepository) ListBySpec(specID string) ([]Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Task
	for _, task := range r.tasks {
		if task.SpecID == specID {
			result = append(result, *task)
		}
	}
	return result, nil
}
