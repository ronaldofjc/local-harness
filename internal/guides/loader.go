package guides

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	harnessfs "github.com/ronaldofjc/local-harness/internal/harness/fs"
)

// Guide representa um guide carregado do filesystem.
type Guide struct {
	ID      string
	Kind    string // rules, skills, subagents
	Content string
}

// Repository define a interface para acesso a guides.
type Repository interface {
	List(kind string) ([]Guide, error)
	Get(kind, id string) (*Guide, error)
	All() ([]Guide, error)
}

// FileSystemRepository implementa Repository a partir do filesystem.
type FileSystemRepository struct {
	root   string
	mu     sync.RWMutex
	guides map[string]*Guide // chave: kind/id
}

// NewFileSystemRepository cria um novo repositorio de guides.
func NewFileSystemRepository(root string) *FileSystemRepository {
	return &FileSystemRepository{
		root:   root,
		guides: make(map[string]*Guide),
	}
}

// Load carrega todos os guides do diretorio guides/.
func (r *FileSystemRepository) Load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	clear(r.guides)
	guidesDir := filepath.Join(r.root, "guides")

	for _, kind := range []string{"rules", "skills", "subagents"} {
		dir := filepath.Join(guidesDir, kind)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("read guides/%s: %w", kind, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			id := strings.TrimSuffix(entry.Name(), ".md")
			path := filepath.Join(dir, entry.Name())
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read guide %s: %w", path, err)
			}
			key := fmt.Sprintf("%s/%s", kind, id)
			r.guides[key] = &Guide{
				ID:      id,
				Kind:    kind,
				Content: string(content),
			}
		}
	}
	return nil
}

// List lista guides por kind.
func (r *FileSystemRepository) List(kind string) ([]Guide, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Guide
	for key, g := range r.guides {
		if kind == "" || strings.HasPrefix(key, kind+"/") {
			result = append(result, *g)
		}
	}
	return result, nil
}

// Get retorna um guide especifico.
func (r *FileSystemRepository) Get(kind, id string) (*Guide, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := fmt.Sprintf("%s/%s", kind, id)
	g, ok := r.guides[key]
	if !ok {
		return nil, fmt.Errorf("guide not found: %s", key)
	}
	return g, nil
}

// All retorna todos os guides.
func (r *FileSystemRepository) All() ([]Guide, error) {
	return r.List("")
}

// StartWatcher inicia o watcher para recarregar guides automaticamente.
func (r *FileSystemRepository) StartWatcher() error {
	w, err := harnessfs.NewWatcher(r.root)
	if err != nil {
		return err
	}
	w.OnEvent(func(ev harnessfs.Event) {
		if strings.Contains(ev.Path, "/guides/") && strings.HasSuffix(ev.Path, ".md") {
			_ = r.Load()
		}
	})
	return nil
}
