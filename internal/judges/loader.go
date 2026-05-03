package judges

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

// Rubric representa uma rubrica de judge.
type Rubric struct {
	ID          string            `yaml:"id"`
	Regulation  common.Regulation `yaml:"regulation"`
	Description string            `yaml:"description"`
	Inputs      []RubricInput     `yaml:"inputs,omitempty"`
	Prompt      string            // corpo do prompt sem frontmatter
	Schema      string            // conteudo JSON do schema
}

// RubricInput define um input esperado pela rubric.
type RubricInput struct {
	Kind     string `yaml:"kind"`
	Optional bool   `yaml:"optional"`
}

// Repository define a interface para acesso a rubrics.
type Repository interface {
	List(filterID string) ([]Rubric, error)
	Get(id string) (*Rubric, error)
}

// FileSystemRepository implementa Repository a partir do filesystem.
type FileSystemRepository struct {
	root    string
	mu      sync.RWMutex
	rubrics map[string]*Rubric
}

// NewFileSystemRepository cria um novo repositorio de rubrics.
func NewFileSystemRepository(root string) *FileSystemRepository {
	return &FileSystemRepository{
		root:    root,
		rubrics: make(map[string]*Rubric),
	}
}

// Load carrega todas as rubrics do diretorio judges/.
func (r *FileSystemRepository) Load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	clear(r.rubrics)
	judgesDir := filepath.Join(r.root, "judges")

	entries, err := os.ReadDir(judgesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read judges dir: %w", err)
	}

	// Agrupa arquivos por rubric_id
	mdFiles := make(map[string]string)
	schemaFiles := make(map[string]string)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		path := filepath.Join(judgesDir, name)

		if strings.HasSuffix(name, ".schema.json") {
			id := strings.TrimSuffix(name, ".schema.json")
			schemaFiles[id] = path
		} else if strings.HasSuffix(name, ".md") {
			id := strings.TrimSuffix(name, ".md")
			mdFiles[id] = path
		}
	}

	// Carrega pares
	for id, mdPath := range mdFiles {
		rubric, err := r.loadRubric(id, mdPath, schemaFiles[id])
		if err != nil {
			return fmt.Errorf("load rubric %s: %w", id, err)
		}
		r.rubrics[id] = rubric
	}

	return nil
}

func (r *FileSystemRepository) loadRubric(id, mdPath, schemaPath string) (*Rubric, error) {
	data, err := os.ReadFile(mdPath)
	if err != nil {
		return nil, err
	}

	content := string(data)
	var rubric Rubric

	// Extrai frontmatter YAML
	if strings.HasPrefix(content, "---") {
		end := strings.Index(content[3:], "---")
		if end != -1 {
			frontmatter := content[3 : end+3]
			if err := yaml.Unmarshal([]byte(frontmatter), &rubric); err != nil {
				return nil, fmt.Errorf("parse frontmatter: %w", err)
			}
			rubric.Prompt = strings.TrimSpace(content[end+6:])
		}
	} else {
		rubric.Prompt = content
	}

	rubric.ID = id

	if schemaPath != "" {
		schemaData, err := os.ReadFile(schemaPath)
		if err != nil {
			return nil, fmt.Errorf("read schema: %w", err)
		}
		rubric.Schema = string(schemaData)
	}

	return &rubric, nil
}

// List lista rubrics com filtro opcional por ID.
func (r *FileSystemRepository) List(filterID string) ([]Rubric, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Rubric
	for id, rubric := range r.rubrics {
		if filterID != "" && id != filterID {
			continue
		}
		result = append(result, *rubric)
	}
	return result, nil
}

// Get retorna uma rubric pelo ID.
func (r *FileSystemRepository) Get(id string) (*Rubric, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rubric, ok := r.rubrics[id]
	if !ok {
		return nil, fmt.Errorf("rubric not found: %s", id)
	}
	return rubric, nil
}

// StartWatcher inicia o watcher para recarregar rubrics automaticamente.
func (r *FileSystemRepository) StartWatcher() (*harnessfs.Watcher, error) {
	w, err := harnessfs.NewWatcher(r.root)
	if err != nil {
		return nil, err
	}
	w.OnEvent(func(ev harnessfs.Event) {
		if strings.Contains(ev.Path, "/judges/") {
			_ = r.Load()
		}
	})
	return w, nil
}
