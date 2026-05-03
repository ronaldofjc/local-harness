package contracts

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

// Spec representa uma especificacao comportamental.
type Spec struct {
	ID                 string     `yaml:"id"`
	Title              string     `yaml:"title"`
	AcceptanceCriteria []string   `yaml:"acceptanceCriteria"`
	Checks             []Check    `yaml:"checks"`
	Tasks              []SpecTask `yaml:"tasks"`
}

// Check define um sensor ou judge a ser executado na validacao.
type Check struct {
	Kind   string `yaml:"kind"` // sensor | judge
	ID     string `yaml:"id"`   // sensor_id ou rubric_id
	Rubric string `yaml:"rubric,omitempty"`
}

// SpecTask define uma task dentro de uma spec.
type SpecTask struct {
	ID             string `yaml:"id"`
	Description    string `yaml:"description"`
	AcceptanceRefs []int  `yaml:"acceptanceRefs"`
}

// SpecRepository define a interface para acesso a specs.
type SpecRepository interface {
	Get(id string) (*Spec, error)
}

// FileSystemSpecRepository implementa SpecRepository a partir do filesystem.
type FileSystemSpecRepository struct {
	root  string
	mu    sync.RWMutex
	specs map[string]*Spec
}

// NewFileSystemSpecRepository cria um novo repositorio de specs.
func NewFileSystemSpecRepository(root string) *FileSystemSpecRepository {
	return &FileSystemSpecRepository{
		root:  root,
		specs: make(map[string]*Spec),
	}
}

// Load carrega todas as specs do diretorio contracts/specs/.
func (r *FileSystemSpecRepository) Load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	clear(r.specs)
	specsDir := filepath.Join(r.root, "contracts", "specs")

	entries, err := os.ReadDir(specsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read specs dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".md") {
			continue
		}

		path := filepath.Join(specsDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read spec %s: %w", path, err)
		}

		var spec Spec
		if err := yaml.Unmarshal(data, &spec); err != nil {
			// TODO: suportar markdown com frontmatter
			return fmt.Errorf("parse spec %s: %w", path, err)
		}

		// Valida que ID do arquivo bate com o ID interno
		fileID := strings.TrimSuffix(name, filepath.Ext(name))
		if spec.ID != "" && spec.ID != fileID {
			return fmt.Errorf("%w: file %s has id %s", common.ErrSpecParseError, name, spec.ID)
		}
		if spec.ID == "" {
			spec.ID = fileID
		}

		r.specs[spec.ID] = &spec
	}
	return nil
}

// Get retorna uma spec pelo ID.
func (r *FileSystemSpecRepository) Get(id string) (*Spec, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	spec, ok := r.specs[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", common.ErrSpecNotFound, id)
	}
	return spec, nil
}

// StartWatcher inicia o watcher para recarregar specs automaticamente.
func (r *FileSystemSpecRepository) StartWatcher() (*harnessfs.Watcher, error) {
	w, err := harnessfs.NewWatcher(r.root)
	if err != nil {
		return nil, err
	}
	w.OnEvent(func(ev harnessfs.Event) {
		if strings.Contains(ev.Path, "/contracts/specs/") {
			_ = r.Load()
		}
	})
	return w, nil
}
