package harnessfs

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Watcher observa mudancas no filesystem sob um diretorio raiz.
type Watcher struct {
	root     string
	watcher  *fsnotify.Watcher
	events   chan Event
	stop     chan struct{}
	mu       sync.RWMutex
	handlers []EventHandler
}

// Event representa uma mudanca observada.
type Event struct {
	Op   string // create, write, remove, rename
	Path string
}

// EventHandler e chamado quando um evento e detectado.
type EventHandler func(Event)

// NewWatcher cria um novo watcher para o diretorio root.
func NewWatcher(root string) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		root:    root,
		watcher: fw,
		events:  make(chan Event, 100),
		stop:    make(chan struct{}),
	}

	if err := w.addRecursive(root); err != nil {
		fw.Close()
		return nil, err
	}

	go w.loop()
	return w, nil
}

// OnEvent registra um handler para eventos.
func (w *Watcher) OnEvent(h EventHandler) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.handlers = append(w.handlers, h)
}

// Close para o watcher.
func (w *Watcher) Close() error {
	close(w.stop)
	return w.watcher.Close()
}

func (w *Watcher) addRecursive(dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return w.watcher.Add(path)
		}
		return nil
	})
}

func (w *Watcher) loop() {
	for {
		select {
		case ev, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			var op string
			switch {
			case ev.Op&fsnotify.Create == fsnotify.Create:
				op = "create"
				// Se for diretorio novo, adiciona ao watcher
				if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
					w.watcher.Add(ev.Name)
				}
			case ev.Op&fsnotify.Write == fsnotify.Write:
				op = "write"
			case ev.Op&fsnotify.Remove == fsnotify.Remove:
				op = "remove"
			case ev.Op&fsnotify.Rename == fsnotify.Rename:
				op = "rename"
			default:
				continue
			}
			e := Event{Op: op, Path: ev.Name}
			w.dispatch(e)
		case _, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			// loga erro silenciosamente em producao
		case <-w.stop:
			return
		}
	}
}

func (w *Watcher) dispatch(e Event) {
	w.mu.RLock()
	handlers := make([]EventHandler, len(w.handlers))
	copy(handlers, w.handlers)
	w.mu.RUnlock()
	for _, h := range handlers {
		h(e)
	}
}

// HarnessRoot retorna o diretorio .harness a partir da variavel de ambiente ou busca
// o .harness mais proximo subindo na arvore de diretorios.
func HarnessRoot() string {
	if root := os.Getenv("HARNESS_ROOT"); root != "" {
		return root
	}
	dir, err := os.Getwd()
	if err != nil {
		return ".harness"
	}
	for {
		candidate := filepath.Join(dir, ".harness")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ".harness"
}
