package sessions

import (
	"encoding/json"
	"time"
)

// SessionHeader representa o cabecalho de uma sessao (primeira linha do jsonl).
type SessionHeader struct {
	SessionID  string    `json:"session_id"`
	Workflow   string    `json:"workflow,omitempty"`
	ContractID string    `json:"contract_id,omitempty"`
	StartedAt  time.Time `json:"startedAt"`
}

// SessionEvent representa um evento dentro de uma sessao.
type SessionEvent struct {
	Type      string          `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// EventType define os tipos de eventos suportados.
type EventType string

const (
	EventTypeToolCall          EventType = "tool_call"
	EventTypeSensorRun         EventType = "sensor_run"
	EventTypeJudgeReview       EventType = "judge_review"
	EventTypeDecision          EventType = "decision"
	EventTypeHumanIntervention EventType = "human_intervention"
)
