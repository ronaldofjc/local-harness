package workflows

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Workflow representa um workflow carregado do filesystem.
type Workflow struct {
	ID          string
	Name        string
	Description string
	Content     string
}

// Repository define a interface para acesso a workflows.
type Repository interface {
	Get(id string) (*Workflow, error)
	All() ([]Workflow, error)
}

// FileSystemRepository implementa Repository a partir do filesystem.
type FileSystemRepository struct {
	root      string
	mu        sync.RWMutex
	workflows map[string]*Workflow
}

// NewFileSystemRepository cria um novo repositorio de workflows.
func NewFileSystemRepository(root string) *FileSystemRepository {
	return &FileSystemRepository{
		root:      root,
		workflows: make(map[string]*Workflow),
	}
}

// Load carrega todos os workflows do diretorio workflows/.
func (r *FileSystemRepository) Load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	clear(r.workflows)
	workflowsDir := filepath.Join(r.root, "workflows")

	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read workflows dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}

		path := filepath.Join(workflowsDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read workflow %s: %w", path, err)
		}

		id := strings.TrimSuffix(name, ".md")
		wf := &Workflow{
			ID:          id,
			Name:        id,
			Description: fmt.Sprintf("Workflow: %s", id),
			Content:     string(data),
		}

		// Tenta extrair titulo da primeira linha se for h1
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "# ") {
				wf.Name = strings.TrimPrefix(line, "# ")
				break
			}
			if line != "" {
				break
			}
		}

		r.workflows[id] = wf
	}

	return nil
}

// Get retorna um workflow pelo ID.
func (r *FileSystemRepository) Get(id string) (*Workflow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	wf, ok := r.workflows[id]
	if !ok {
		return nil, fmt.Errorf("workflow not found: %s", id)
	}
	return wf, nil
}

// All retorna todos os workflows carregados.
func (r *FileSystemRepository) All() ([]Workflow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Workflow
	for _, wf := range r.workflows {
		result = append(result, *wf)
	}
	return result, nil
}
