package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ronaldofjc/local-harness/internal/common"
	"github.com/ronaldofjc/local-harness/internal/contracts"
	"github.com/ronaldofjc/local-harness/internal/guides"
	"github.com/ronaldofjc/local-harness/internal/judges"
	"github.com/ronaldofjc/local-harness/internal/sensors"
	"github.com/ronaldofjc/local-harness/internal/sessions"
	"github.com/ronaldofjc/local-harness/internal/steering"
	"github.com/ronaldofjc/local-harness/internal/workflows"
)

// Server representa o servidor MCP.
type Server struct {
	version          string
	guidesRepo       guides.Repository
	sensorsService   *sensors.Service
	judgesService    *judges.Service
	contractsService *contracts.Service
	sessionsService  *sessions.Service
	steeringService  *steering.Service
	workflowsRepo    workflows.Repository
}

// NewServer cria uma nova instancia do servidor MCP.
func NewServer(version string, guidesRepo guides.Repository, sensorsService *sensors.Service, judgesService *judges.Service, contractsService *contracts.Service, sessionsService *sessions.Service, steeringService *steering.Service, workflowsRepo workflows.Repository) *Server {
	return &Server{
		version:          version,
		guidesRepo:       guidesRepo,
		sensorsService:   sensorsService,
		judgesService:    judgesService,
		contractsService: contractsService,
		sessionsService:  sessionsService,
		steeringService:  steeringService,
		workflowsRepo:    workflowsRepo,
	}
}

// Handle processa uma requisicao JSON-RPC e retorna a resposta.
func (s *Server) Handle(req JSONRPCRequest) JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	case "resources/list":
		return s.handleResourcesList(req)
	case "resources/read":
		return s.handleResourcesRead(req)
	case "prompts/list":
		return s.handlePromptsList(req)
	case "prompts/get":
		return s.handlePromptsGet(req)
	default:
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32601,
				Message: fmt.Sprintf("method not found: %s", req.Method),
			},
		}
	}
}

func (s *Server) handleInitialize(req JSONRPCRequest) JSONRPCResponse {
	var initReq InitializeRequest
	if err := json.Unmarshal(req.Params, &initReq); err != nil {
		return errorResponse(req.ID, -32700, "parse error")
	}

	result := InitializeResult{
		ProtocolVersion: initReq.ProtocolVersion,
		Capabilities: ServerCapabilities{
			Tools:     &ToolsCapability{ListChanged: true},
			Resources: &ResourcesCapability{ListChanged: true, Subscribe: true},
			Prompts:   &PromptsCapability{ListChanged: true},
		},
		ServerInfo: struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}{
			Name:    "local-harness",
			Version: s.version,
		},
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

func (s *Server) handleToolsList(req JSONRPCRequest) JSONRPCResponse {
	tools := []Tool{
		// Sensors
		{
			Name:        "sensor.list",
			Description: "Lista todos os sensores registrados em .harness/sensors/ com filtros opcionais por kind e regulation.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"kind":{"type":"string"},"regulation":{"type":"string"}},"required":[]}`),
		},
		{
			Name:        "sensor.run",
			Description: "Executa um sensor pelo ID e retorna o output normalizado.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"},"target":{"type":"string"}},"required":["id"]}`),
		},
		{
			Name:        "sensor.register",
			Description: "Adiciona ou atualiza um sensor em .harness/sensors/.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"},"kind":{"type":"string"},"regulation":{"type":"string"},"command":{"type":"string"},"adapter":{"type":"string"},"description":{"type":"string"},"defaults":{"type":"object"}},"required":["id","kind","regulation","command","adapter"]}`),
		},
		// Judges
		{
			Name:        "judge.list",
			Description: "Lista rubrics de judges disponiveis.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"rubric":{"type":"string"}},"required":[]}`),
		},
		{
			Name:        "judge.review",
			Description: "Renderiza prompt + schema + contexto para o cliente MCP avaliar (fase 1).",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"rubric_id":{"type":"string"},"target":{"type":"string"},"spec_id":{"type":"string"}},"required":["rubric_id","target"]}`),
		},
		{
			Name:        "judge.record",
			Description: "Recebe o verdict do cliente, valida pelo schema e retorna envelope normalizado (fase 2).",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"rubric_id":{"type":"string"},"target":{"type":"string"},"spec_id":{"type":"string"},"result":{"type":"object"}},"required":["rubric_id","target","result"]}`),
		},
		// Contracts
		{
			Name:        "contract.spec.validate",
			Description: "Valida uma spec contra um artefato, orquestrando sensors e judges.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"},"artifact":{"type":"string"}},"required":["id","artifact"]}`),
		},
		{
			Name:        "contract.task.next",
			Description: "Retorna a proxima task pendente de uma spec.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"spec_id":{"type":"string"}},"required":["spec_id"]}`),
		},
		{
			Name:        "contract.task.complete",
			Description: "Marca uma task como completed com evidencias.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"},"evidence":{"type":"array"}},"required":["task_id"]}`),
		},
		// Sessions
		{
			Name:        "session.start",
			Description: "Inicia uma nova sessao append-only em .harness/.local/sessions/. Reutiliza a sessao ativa da janela de execucao, a menos que force_new seja true.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"workflow":{"type":"string"},"contract_id":{"type":"string"},"force_new":{"type":"boolean"}},"required":[]}`),
		},
		{
			Name:        "session.append",
			Description: "Adiciona um evento a uma sessao existente.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"session_id":{"type":"string"},"event":{"type":"object"}},"required":["session_id","event"]}`),
		},
		{
			Name:        "session.get",
			Description: "Le o cabecalho e todos os eventos de uma sessao.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"session_id":{"type":"string"}},"required":["session_id"]}`),
		},
		// Steering
		{
			Name:        "harness.steer.suggest",
			Description: "Analisa o steering log e sugere novos guides baseado em padroes de violations.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"windowDays":{"type":"integer"}},"required":[]}`),
		},
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"tools": tools,
		},
	}
}

func (s *Server) handleToolsCall(req JSONRPCRequest) JSONRPCResponse {
	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &call); err != nil {
		return errorResponse(req.ID, -32700, "parse error")
	}

	// Log de tool_call no steering com session_id
	activeSessionID := s.sessionsService.ActiveSessionID()
	if s.steeringService != nil {
		var argsMap map[string]any
		_ = json.Unmarshal(call.Arguments, &argsMap)
		_ = s.steeringService.Log().Append(steering.Event{
			Timestamp: time.Now().UTC(),
			SessionID: activeSessionID,
			Source:    "tool_call",
			Tool:      call.Name,
			Args:      argsMap,
		})
	}

	switch call.Name {
	// Sensors
	case "sensor.list":
		return s.handleSensorList(req.ID, call.Arguments)
	case "sensor.run":
		return s.handleSensorRun(req.ID, call.Arguments)
	case "sensor.register":
		return s.handleSensorRegister(req.ID, call.Arguments)
	// Judges
	case "judge.list":
		return s.handleJudgeList(req.ID, call.Arguments)
	case "judge.review":
		return s.handleJudgeReview(req.ID, call.Arguments)
	case "judge.record":
		return s.handleJudgeRecord(req.ID, call.Arguments)
	// Contracts
	case "contract.spec.validate":
		return s.handleContractValidate(req.ID, call.Arguments)
	case "contract.task.next":
		return s.handleContractTaskNext(req.ID, call.Arguments)
	case "contract.task.complete":
		return s.handleContractTaskComplete(req.ID, call.Arguments)
	// Sessions
	case "session.start":
		return s.handleSessionStart(req.ID, call.Arguments)
	case "session.append":
		return s.handleSessionAppend(req.ID, call.Arguments)
	case "session.get":
		return s.handleSessionGet(req.ID, call.Arguments)
	// Steering
	case "harness.steer.suggest":
		return s.handleSteerSuggest(req.ID, call.Arguments)
	default:
		return errorResponse(req.ID, -32601, fmt.Sprintf("tool not found: %s", call.Name))
	}
}

// --- Sensor handlers ---

func (s *Server) handleSensorList(id any, args json.RawMessage) JSONRPCResponse {
	var input struct {
		Kind       string `json:"kind,omitempty"`
		Regulation string `json:"regulation,omitempty"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return errorResponse(id, -32700, "parse error")
	}

	reg := common.Regulation(input.Regulation)
	sensorList, err := s.sensorsService.List(input.Kind, reg)
	if err != nil {
		return errorResponse(id, -32603, err.Error())
	}

	result := make([]map[string]any, len(sensorList))
	for i, s := range sensorList {
		result[i] = map[string]any{
			"id":          s.ID,
			"kind":        s.Kind,
			"regulation":  s.Regulation,
			"command":     s.Command,
			"description": s.Description,
		}
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]any{
			"rubrics": result,
		},
	}
}

func (s *Server) handleSensorRun(id any, args json.RawMessage) JSONRPCResponse {
	var input struct {
		ID     string `json:"id"`
		Target string `json:"target,omitempty"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return errorResponse(id, -32700, "parse error")
	}

	output, err := s.sensorsService.Run(input.ID, input.Target)
	if err != nil {
		return errorResponse(id, -32603, err.Error())
	}

	activeSessionID := s.sessionsService.ActiveSessionID()

	// Loga no steering log com rastreabilidade
	if s.steeringService != nil {
		_ = s.steeringService.Log().Append(steering.Event{
			Timestamp:  time.Now().UTC(),
			SessionID:  activeSessionID,
			Source:     "sensor",
			Tool:       input.ID,
			Regulation: output.Regulation,
			Passed:     output.Passed,
			Violations: output.Violations,
		})
	}

	// Auto-append na sessão ativa
	if activeSessionID != "" {
		payload, _ := json.Marshal(map[string]any{
			"sensor":     input.ID,
			"passed":     output.Passed,
			"regulation": output.Regulation,
		})
		eventData, _ := json.Marshal(sessions.SessionEvent{
			Type:      "sensor_run",
			Timestamp: time.Now().UTC(),
			Payload:   payload,
		})
		_, _ = s.sessionsService.Append(sessions.AppendInput{
			SessionID: activeSessionID,
			Event:     eventData,
		})
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  output,
	}
}

func (s *Server) handleSensorRegister(id any, args json.RawMessage) JSONRPCResponse {
	var input sensors.Sensor
	if err := json.Unmarshal(args, &input); err != nil {
		return errorResponse(id, -32700, "parse error")
	}

	if err := s.sensorsService.Register(input); err != nil {
		return errorResponse(id, -32603, err.Error())
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]any{
			"registered": true,
			"path":       fmt.Sprintf(".harness/sensors/%s.yaml", input.ID),
			"changed":    true,
		},
	}
}

// --- Judge handlers ---

func (s *Server) handleJudgeList(id any, args json.RawMessage) JSONRPCResponse {
	var input struct {
		Rubric string `json:"rubric,omitempty"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return errorResponse(id, -32700, "parse error")
	}

	rubrics, err := s.judgesService.List(input.Rubric)
	if err != nil {
		return errorResponse(id, -32603, err.Error())
	}

	result := make([]map[string]any, len(rubrics))
	for i, r := range rubrics {
		result[i] = map[string]any{
			"rubric_id":   r.ID,
			"regulation":  r.Regulation,
			"description": r.Description,
		}
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]any{
			"rubrics": result,
		},
	}
}

func (s *Server) handleJudgeReview(id any, args json.RawMessage) JSONRPCResponse {
	var input judges.ReviewInput
	if err := json.Unmarshal(args, &input); err != nil {
		return errorResponse(id, -32700, "parse error")
	}

	output, err := s.judgesService.Review(input)
	if err != nil {
		return errorResponse(id, -32603, err.Error())
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  output,
	}
}

func (s *Server) handleJudgeRecord(id any, args json.RawMessage) JSONRPCResponse {
	var input judges.RecordInput
	if err := json.Unmarshal(args, &input); err != nil {
		return errorResponse(id, -32700, "parse error")
	}

	output, err := s.judgesService.Record(input)
	if err != nil {
		return errorResponse(id, -32603, err.Error())
	}

	activeSessionID := s.sessionsService.ActiveSessionID()

	// Loga no steering log com rastreabilidade
	if s.steeringService != nil {
		_ = s.steeringService.Log().Append(steering.Event{
			Timestamp:  time.Now().UTC(),
			SessionID:  activeSessionID,
			SpecID:     input.SpecID,
			Source:     "judge",
			Tool:       input.RubricID,
			Regulation: output.Regulation,
			Passed:     output.Passed,
			Violations: output.Violations,
		})
	}

	// Auto-append na sessão ativa
	if activeSessionID != "" {
		payload, _ := json.Marshal(map[string]any{
			"rubric":  input.RubricID,
			"passed":  output.Passed,
			"spec_id": input.SpecID,
		})
		eventData, _ := json.Marshal(sessions.SessionEvent{
			Type:      "judge_review",
			Timestamp: time.Now().UTC(),
			Payload:   payload,
		})
		_, _ = s.sessionsService.Append(sessions.AppendInput{
			SessionID: activeSessionID,
			Event:     eventData,
		})
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  output,
	}
}

// --- Contract handlers ---

func (s *Server) handleContractValidate(id any, args json.RawMessage) JSONRPCResponse {
	var input contracts.ValidateInput
	if err := json.Unmarshal(args, &input); err != nil {
		return errorResponse(id, -32700, "parse error")
	}

	output, err := s.contractsService.Validate(input)
	if err != nil {
		return errorResponse(id, -32603, err.Error())
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  output,
	}
}

func (s *Server) handleContractTaskNext(id any, args json.RawMessage) JSONRPCResponse {
	var input contracts.TaskNextInput
	if err := json.Unmarshal(args, &input); err != nil {
		return errorResponse(id, -32700, "parse error")
	}

	output, err := s.contractsService.TaskNext(input)
	if err != nil {
		return errorResponse(id, -32603, err.Error())
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  output,
	}
}

func (s *Server) handleContractTaskComplete(id any, args json.RawMessage) JSONRPCResponse {
	var input contracts.TaskCompleteInput
	if err := json.Unmarshal(args, &input); err != nil {
		return errorResponse(id, -32700, "parse error")
	}

	output, err := s.contractsService.TaskComplete(input)
	if err != nil {
		return errorResponse(id, -32603, err.Error())
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  output,
	}
}

// --- Resource handlers ---

func (s *Server) handleResourcesList(req JSONRPCRequest) JSONRPCResponse {
	allGuides, err := s.guidesRepo.All()
	if err != nil {
		return errorResponse(req.ID, -32603, err.Error())
	}

	resources := make([]Resource, 0, len(allGuides))
	for _, g := range allGuides {
		resources = append(resources, Resource{
			URI:         fmt.Sprintf("harness://guides/%s/%s", g.Kind, g.ID),
			Name:        g.ID,
			Description: fmt.Sprintf("Guide: %s/%s", g.Kind, g.ID),
			MIMEType:    "text/markdown",
		})
	}

	// Adiciona workflows como fallback resources
	if s.workflowsRepo != nil {
		allWorkflows, err := s.workflowsRepo.All()
		if err == nil {
			for _, wf := range allWorkflows {
				resources = append(resources, Resource{
					URI:         fmt.Sprintf("harness://workflows/%s", wf.ID),
					Name:        wf.ID,
					Description: fmt.Sprintf("Workflow: %s", wf.Name),
					MIMEType:    "text/markdown",
				})
			}
		}
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"resources": resources,
		},
	}
}

func (s *Server) handleResourcesRead(req JSONRPCRequest) JSONRPCResponse {
	var input struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(req.Params, &input); err != nil {
		return errorResponse(req.ID, -32700, "parse error")
	}

	// Parse URI: harness://guides/{kind}/{id}
	if strings.HasPrefix(input.URI, "harness://guides/") {
		parts := strings.Split(strings.TrimPrefix(input.URI, "harness://guides/"), "/")
		if len(parts) != 2 {
			return errorResponse(req.ID, -32602, "invalid resource URI")
		}

		kind, id := parts[0], parts[1]
		guide, err := s.guidesRepo.Get(kind, id)
		if err != nil {
			return errorResponse(req.ID, -32602, err.Error())
		}

		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"contents": []ResourceContents{{
					URI:      input.URI,
					MIMEType: "text/markdown",
					Text:     guide.Content,
				}},
			},
		}
	}

	// Parse URI: harness://workflows/{id}
	if strings.HasPrefix(input.URI, "harness://workflows/") {
		id := strings.TrimPrefix(input.URI, "harness://workflows/")
		if s.workflowsRepo == nil {
			return errorResponse(req.ID, -32602, "workflows not available")
		}
		wf, err := s.workflowsRepo.Get(id)
		if err != nil {
			return errorResponse(req.ID, -32602, err.Error())
		}

		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"contents": []ResourceContents{{
					URI:      input.URI,
					MIMEType: "text/markdown",
					Text:     wf.Content,
				}},
			},
		}
	}

	return errorResponse(req.ID, -32602, "invalid resource URI")
}

func (s *Server) handlePromptsList(req JSONRPCRequest) JSONRPCResponse {
	var allWorkflows []workflows.Workflow
	if s.workflowsRepo != nil {
		wfs, err := s.workflowsRepo.All()
		if err == nil {
			allWorkflows = wfs
		}
	}
	promptInfos := workflows.ToPrompts(allWorkflows)
	prompts := make([]Prompt, 0, len(promptInfos))
	for _, pi := range promptInfos {
		prompts = append(prompts, Prompt{
			Name:        pi.Name,
			Description: pi.Description,
		})
	}
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"prompts": prompts,
		},
	}
}

func (s *Server) handlePromptsGet(req JSONRPCRequest) JSONRPCResponse {
	var input struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(req.Params, &input); err != nil {
		return errorResponse(req.ID, -32700, "parse error")
	}

	if s.workflowsRepo == nil {
		return errorResponse(req.ID, -32602, "prompts not available")
	}

	// Remove prefixo "workflow." se presente
	id := strings.TrimPrefix(input.Name, "workflow.")
	wf, err := s.workflowsRepo.Get(id)
	if err != nil {
		return errorResponse(req.ID, -32602, err.Error())
	}

	msgInfos := workflows.GetPromptMessages(wf)
	messages := make([]PromptMessage, 0, len(msgInfos))
	for _, mi := range msgInfos {
		messages = append(messages, PromptMessage{
			Role:    mi.Role,
			Content: mi.Content,
		})
	}
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"description": wf.Description,
			"messages":    messages,
		},
	}
}

// --- Session handlers ---

func (s *Server) handleSessionStart(id any, args json.RawMessage) JSONRPCResponse {
	var input sessions.StartInput
	if err := json.Unmarshal(args, &input); err != nil {
		return errorResponse(id, -32700, "parse error")
	}

	output, err := s.sessionsService.Start(input)
	if err != nil {
		return errorResponse(id, -32603, err.Error())
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  output,
	}
}

func (s *Server) handleSessionAppend(id any, args json.RawMessage) JSONRPCResponse {
	var input sessions.AppendInput
	if err := json.Unmarshal(args, &input); err != nil {
		return errorResponse(id, -32700, "parse error")
	}

	output, err := s.sessionsService.Append(input)
	if err != nil {
		return errorResponse(id, -32603, err.Error())
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  output,
	}
}

func (s *Server) handleSessionGet(id any, args json.RawMessage) JSONRPCResponse {
	var input sessions.GetInput
	if err := json.Unmarshal(args, &input); err != nil {
		return errorResponse(id, -32700, "parse error")
	}

	output, err := s.sessionsService.Get(input)
	if err != nil {
		return errorResponse(id, -32603, err.Error())
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  output,
	}
}

// --- Steering handlers ---

func (s *Server) handleSteerSuggest(id any, args json.RawMessage) JSONRPCResponse {
	var input steering.SuggestInput
	if err := json.Unmarshal(args, &input); err != nil {
		return errorResponse(id, -32700, "parse error")
	}

	output, err := s.steeringService.Suggest(input)
	if err != nil {
		return errorResponse(id, -32603, err.Error())
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  output,
	}
}

func errorResponse(id any, code int, message string) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
	}
}
