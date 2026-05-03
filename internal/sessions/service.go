package sessions

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// sessionMaxAge define o tempo máximo de retenção de sessões antigas.
const sessionMaxAge = 7 * 24 * time.Hour

// Service orquestra operações de sessões.
type Service struct {
	store           *Store
	mu              sync.Mutex
	activeSessionID string
}

// NewService cria um novo service de sessões.
func NewService(store *Store) *Service {
	return &Service{store: store}
}

// StartInput define os parâmetros de entrada para session.start.
type StartInput struct {
	Workflow   string `json:"workflow,omitempty"`
	ContractID string `json:"contract_id,omitempty"`
	ForceNew   bool   `json:"force_new,omitempty"`
}

// StartOutput define a resposta de session.start.
type StartOutput struct {
	SessionID string `json:"session_id"`
	StartedAt string `json:"startedAt"`
	Reused    bool   `json:"reused"`
}

// Start cria uma nova sessão ou reutiliza a sessão ativa da janela de execução.
func (s *Service) Start(input StartInput) (*StartOutput, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Housekeeping: limpa sessões com mais de 7 dias.
	// Ignora erro de housekeeping para não bloquear a criação de sessão.
	_, _ = s.store.DeleteOldSessions(sessionMaxAge)

	// Reutiliza sessão ativa se existir, não foi forçada nova e arquivo ainda está presente.
	if !input.ForceNew && s.activeSessionID != "" && s.store.Exists(s.activeSessionID) {
		header, _, err := s.store.Get(s.activeSessionID)
		if err == nil {
			return &StartOutput{
				SessionID: s.activeSessionID,
				StartedAt: header.StartedAt.Format(time.RFC3339),
				Reused:    true,
			}, nil
		}
		// Se falhou em ler, limpa referência e segue para criar nova.
		s.activeSessionID = ""
	}

	// Cria nova sessão.
	sessionID, err := s.store.Start(input.Workflow, input.ContractID)
	if err != nil {
		return nil, err
	}
	s.activeSessionID = sessionID

	header, _, err := s.store.Get(sessionID)
	if err != nil {
		return nil, err
	}

	return &StartOutput{
		SessionID: sessionID,
		StartedAt: header.StartedAt.Format(time.RFC3339),
		Reused:    false,
	}, nil
}

// AppendInput define os parâmetros de entrada para session.append.
type AppendInput struct {
	SessionID string          `json:"session_id"`
	Event     json.RawMessage `json:"event"`
}

// AppendOutput define a resposta de session.append.
type AppendOutput struct {
	Appended   bool `json:"appended"`
	EventCount int  `json:"eventCount"`
}

// Append adiciona um evento a uma sessão.
func (s *Service) Append(input AppendInput) (*AppendOutput, error) {
	var event SessionEvent
	if err := json.Unmarshal(input.Event, &event); err != nil {
		return nil, fmt.Errorf("invalid event: %w", err)
	}

	if err := s.store.Append(input.SessionID, event); err != nil {
		return nil, err
	}

	_, events, err := s.store.Get(input.SessionID)
	if err != nil {
		return nil, err
	}

	return &AppendOutput{
		Appended:   true,
		EventCount: len(events),
	}, nil
}

// GetInput define os parâmetros de entrada para session.get.
type GetInput struct {
	SessionID string `json:"session_id"`
}

// GetOutput define a resposta de session.get.
type GetOutput struct {
	Header *SessionHeader `json:"header"`
	Events []SessionEvent `json:"events"`
}

// ActiveSessionID retorna a sessão ativa atual (se houver).
func (s *Service) ActiveSessionID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activeSessionID
}

// Get lê uma sessão completa.
func (s *Service) Get(input GetInput) (*GetOutput, error) {
	header, events, err := s.store.Get(input.SessionID)
	if err != nil {
		return nil, err
	}

	return &GetOutput{
		Header: header,
		Events: events,
	}, nil
}
