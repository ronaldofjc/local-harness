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
	Timestamp  time.Time          `json:"timestamp"`
	SessionID  string             `json:"session_id,omitempty"`
	TaskID     string             `json:"task_id,omitempty"`
	SpecID     string             `json:"spec_id,omitempty"`
	Source     string             `json:"source"` // sensor | judge | contract | tool_call
	Tool       string             `json:"tool"`
	Regulation common.Regulation  `json:"regulation,omitempty"`
	Passed     bool               `json:"passed,omitempty"`
	Violations []common.Violation `json:"violations,omitempty"`
	Args       map[string]any     `json:"args,omitempty"`
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
