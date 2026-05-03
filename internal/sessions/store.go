package sessions

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ronaldofjc/local-harness/internal/common"
)

// Store gerencia sessoes append-only em jsonl.
type Store struct {
	root string
}

// NewStore cria um novo store de sessoes.
func NewStore(root string) *Store {
	return &Store{root: filepath.Join(root, ".local", "sessions")}
}

// Start cria uma nova sessao e retorna o ID.
func (s *Store) Start(workflow, contractID string) (string, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}

	if err := os.MkdirAll(s.root, 0755); err != nil {
		return "", fmt.Errorf("mkdir sessions: %w", err)
	}

	header := SessionHeader{
		SessionID:  sessionID,
		Workflow:   workflow,
		ContractID: contractID,
		StartedAt:  time.Now().UTC(),
	}

	path := s.path(sessionID)
	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create session file: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	if err := enc.Encode(header); err != nil {
		return "", fmt.Errorf("encode header: %w", err)
	}

	return sessionID, nil
}

// Append adiciona um evento a uma sessao existente.
func (s *Store) Append(sessionID string, event SessionEvent) error {
	path := s.path(sessionID)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", common.ErrSessionNotFound, sessionID)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open session file: %w", err)
	}
	defer f.Close()

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	enc := json.NewEncoder(f)
	if err := enc.Encode(event); err != nil {
		return fmt.Errorf("encode event: %w", err)
	}

	return nil
}

// Get le o cabecalho e todos os eventos de uma sessao.
func (s *Store) Get(sessionID string) (*SessionHeader, []SessionEvent, error) {
	path := s.path(sessionID)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("%w: %s", common.ErrSessionNotFound, sessionID)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open session file: %w", err)
	}
	defer f.Close()

	var header SessionHeader
	var events []SessionEvent

	scanner := bufio.NewScanner(f)
	first := true
	for scanner.Scan() {
		line := scanner.Bytes()
		if first {
			if err := json.Unmarshal(line, &header); err != nil {
				return nil, nil, fmt.Errorf("parse header: %w", err)
			}
			first = false
			continue
		}

		var event SessionEvent
		if err := json.Unmarshal(line, &event); err != nil {
			return nil, nil, fmt.Errorf("parse event: %w", err)
		}
		events = append(events, event)
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("read session file: %w", err)
	}

	return &header, events, nil
}

// Exists verifica se um arquivo de sessão existe.
func (s *Store) Exists(sessionID string) bool {
	path := s.path(sessionID)
	_, err := os.Stat(path)
	return err == nil
}

// ListSessionIDs retorna todos os IDs de sessão existentes.
func (s *Store) ListSessionIDs() ([]string, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read sessions dir: %w", err)
	}

	var ids []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".jsonl") {
			ids = append(ids, strings.TrimSuffix(name, ".jsonl"))
		}
	}
	return ids, nil
}

// DeleteOldSessions remove arquivos de sessão mais antigos que maxAge.
// Retorna a quantidade de arquivos removidos.
func (s *Store) DeleteOldSessions(maxAge time.Duration) (int, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read sessions dir: %w", err)
	}

	cutoff := time.Now().UTC().Add(-maxAge)
	removed := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			path := filepath.Join(s.root, entry.Name())
			if err := os.Remove(path); err == nil {
				removed++
			}
		}
	}
	return removed, nil
}

func (s *Store) path(sessionID string) string {
	return filepath.Join(s.root, fmt.Sprintf("%s.jsonl", sessionID))
}

func generateSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
