package steering

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ronaldofjc/local-harness/internal/common"
)

// Event representa um evento no steering log.
type Event struct {
	Timestamp  time.Time            `json:"timestamp"`
	Source     string               `json:"source"` // sensor | judge | contract
	Tool       string               `json:"tool"`
	Regulation common.Regulation    `json:"regulation"`
	Passed     bool                 `json:"passed"`
	Violations []common.Violation   `json:"violations,omitempty"`
}

// Log gerencia o steering log append-only.
type Log struct {
	path string
}

// NewLog cria um novo steering log.
func NewLog(root string) *Log {
	return &Log{path: filepath.Join(root, ".local", "steering", "log.jsonl")}
}

// Append adiciona um evento ao log.
func (l *Log) Append(event Event) error {
	dir := filepath.Dir(l.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir steering: %w", err)
	}

	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open steering log: %w", err)
	}
	defer f.Close()

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	enc := json.NewEncoder(f)
	if err := enc.Encode(event); err != nil {
		return fmt.Errorf("encode steering event: %w", err)
	}

	return nil
}

// ReadAll le todos os eventos do log.
func (l *Log) ReadAll() ([]Event, error) {
	if _, err := os.Stat(l.path); os.IsNotExist(err) {
		return []Event{}, nil
	}

	f, err := os.Open(l.path)
	if err != nil {
		return nil, fmt.Errorf("open steering log: %w", err)
	}
	defer f.Close()

	var events []Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue // skip malformed lines
		}
		events = append(events, event)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read steering log: %w", err)
	}

	return events, nil
}
