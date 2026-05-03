package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ronaldofjc/local-harness/internal/contracts"
	"github.com/ronaldofjc/local-harness/internal/guides"
	"github.com/ronaldofjc/local-harness/internal/judges"
	"github.com/ronaldofjc/local-harness/internal/sensors"
	"github.com/ronaldofjc/local-harness/internal/sessions"
	"github.com/ronaldofjc/local-harness/internal/steering"
	"github.com/ronaldofjc/local-harness/internal/workflows"
)

func setupTestServer() *Server {
	root := "../../examples/.harness"

	guidesRepo := guides.NewFileSystemRepository(root)
	_ = guidesRepo.Load()

	sensorsRepo := sensors.NewFileSystemRepository(root)
	_ = sensorsRepo.Load()
	runner := sensors.NewRunner()
	sensorsService := sensors.NewService(sensorsRepo, runner)

	judgesRepo := judges.NewFileSystemRepository(root)
	_ = judgesRepo.Load()
	judgesValidator := judges.NewValidator()
	judgesService := judges.NewService(judgesRepo, judgesValidator)

	specRepo := contracts.NewFileSystemSpecRepository(root)
	_ = specRepo.Load()
	taskRepo := contracts.NewFileSystemTaskRepository(root)
	_ = taskRepo.Load()
	contractsService := contracts.NewService(specRepo, taskRepo, sensorsService)

	sessionsStore := sessions.NewStore(root)
	sessionsService := sessions.NewService(sessionsStore)

	steeringLog := steering.NewLog(root)
	steeringService := steering.NewService(steeringLog)

	workflowsRepo := workflows.NewFileSystemRepository(root)
	_ = workflowsRepo.Load()

	return NewServer("0.1.0", guidesRepo, sensorsService, judgesService, contractsService, sessionsService, steeringService, workflowsRepo)
}

func TestHandleInitialize(t *testing.T) {
	server := setupTestServer()

	params, _ := json.Marshal(InitializeRequest{
		ProtocolVersion: "2024-11-05",
		Capabilities:    map[string]any{},
		ClientInfo: struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}{
			Name:    "test-client",
			Version: "1.0.0",
		},
	})

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  params,
	}

	resp := server.Handle(req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestHandleMethodNotFound(t *testing.T) {
	server := setupTestServer()
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "nonexistent/method",
	}

	resp := server.Handle(req)
	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("expected code -32601, got %d", resp.Error.Code)
	}
}

func TestHandleToolsList(t *testing.T) {
	server := setupTestServer()
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/list",
	}

	resp := server.Handle(req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("expected map result")
	}

	tools, ok := result["tools"].([]Tool)
	if !ok {
		_, ok := result["tools"].([]any)
		if !ok {
			t.Fatalf("expected tools list, got %T", result["tools"])
		}
	}

	_ = tools
}

func TestHandleJudgeReview(t *testing.T) {
	server := setupTestServer()

	params, _ := json.Marshal(map[string]any{
		"name": "judge.review",
		"arguments": map[string]any{
			"rubric_id": "spec-adherence",
			"target":    "internal/foo",
		},
	})

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "tools/call",
		Params:  params,
	}

	resp := server.Handle(req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestHandleContractValidate(t *testing.T) {
	server := setupTestServer()

	params, _ := json.Marshal(map[string]any{
		"name": "contract.spec.validate",
		"arguments": map[string]any{
			"id":       "example-feature",
			"artifact": ".",
		},
	})

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      5,
		Method:  "tools/call",
		Params:  params,
	}

	resp := server.Handle(req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestHandleContractTaskNext(t *testing.T) {
	server := setupTestServer()

	params, _ := json.Marshal(map[string]any{
		"name": "contract.task.next",
		"arguments": map[string]any{
			"spec_id": "example-feature",
		},
	})

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      6,
		Method:  "tools/call",
		Params:  params,
	}

	resp := server.Handle(req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestHandleSessionStart(t *testing.T) {
	server := setupTestServer()

	params, _ := json.Marshal(map[string]any{
		"name": "session.start",
		"arguments": map[string]any{
			"workflow":    "PREVC",
			"contract_id": "test-spec",
		},
	})

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      7,
		Method:  "tools/call",
		Params:  params,
	}

	resp := server.Handle(req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestHandleSessionAppendAndGet(t *testing.T) {
	server := setupTestServer()

	// Start
	startParams, _ := json.Marshal(map[string]any{
		"name":      "session.start",
		"arguments": map[string]any{},
	})
	startReq := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      8,
		Method:  "tools/call",
		Params:  startParams,
	}
	startResp := server.Handle(startReq)
	if startResp.Error != nil {
		t.Fatalf("unexpected error on start: %v", startResp.Error)
	}

	resultJSON, _ := json.Marshal(startResp.Result)
	var resultMap map[string]any
	_ = json.Unmarshal(resultJSON, &resultMap)
	sessionID := resultMap["session_id"].(string)

	// Append
	appendParams, _ := json.Marshal(map[string]any{
		"name": "session.append",
		"arguments": map[string]any{
			"session_id": sessionID,
			"event": map[string]any{
				"type":    "tool_call",
				"payload": map[string]any{"tool": "sensor.run"},
			},
		},
	})
	appendReq := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      9,
		Method:  "tools/call",
		Params:  appendParams,
	}
	appendResp := server.Handle(appendReq)
	if appendResp.Error != nil {
		t.Fatalf("unexpected error on append: %v", appendResp.Error)
	}

	// Get
	getParams, _ := json.Marshal(map[string]any{
		"name": "session.get",
		"arguments": map[string]any{
			"session_id": sessionID,
		},
	})
	getReq := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      10,
		Method:  "tools/call",
		Params:  getParams,
	}
	getResp := server.Handle(getReq)
	if getResp.Error != nil {
		t.Fatalf("unexpected error on get: %v", getResp.Error)
	}
}

func TestHandleSteerSuggest(t *testing.T) {
	server := setupTestServer()

	params, _ := json.Marshal(map[string]any{
		"name": "harness.steer.suggest",
		"arguments": map[string]any{
			"windowDays": 7,
		},
	})

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      11,
		Method:  "tools/call",
		Params:  params,
	}

	resp := server.Handle(req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestHandlePromptsList(t *testing.T) {
	server := setupTestServer()
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      12,
		Method:  "prompts/list",
	}

	resp := server.Handle(req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("expected map result")
	}
	prompts, ok := result["prompts"].([]Prompt)
	if !ok {
		_, ok := result["prompts"].([]any)
		if !ok {
			t.Fatalf("expected prompts list, got %T", result["prompts"])
		}
	}
	_ = prompts
}

func TestHandlePromptsGet(t *testing.T) {
	server := setupTestServer()

	params, _ := json.Marshal(map[string]any{
		"name": "PREVC",
	})

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      13,
		Method:  "prompts/get",
		Params:  params,
	}

	resp := server.Handle(req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestHandleResourcesReadWorkflow(t *testing.T) {
	server := setupTestServer()

	params, _ := json.Marshal(map[string]any{
		"uri": "harness://workflows/PREVC",
	})

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      14,
		Method:  "resources/read",
		Params:  params,
	}

	resp := server.Handle(req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

// TestEndToEnd_Flow simula o ciclo completo de uso do harness:
// sensor.run -> judge.review -> contract.task.next -> contract.task.complete
func TestEndToEnd_Flow(t *testing.T) {
	root := "../../examples/.harness"

	// Limpa tasks de testes anteriores para garantir independencia
	_ = os.RemoveAll(filepath.Join(root, "contracts", "tasks"))

	sensorsRepo := sensors.NewFileSystemRepository(root)
	loadErr := sensorsRepo.Load()
	if loadErr != nil {
		t.Logf("sensors load error: %v", loadErr)
	}
	sensorList, _ := sensorsRepo.List("", "")
	t.Logf("sensors loaded: %d", len(sensorList))
	for _, s := range sensorList {
		t.Logf("  - %s", s.ID)
	}
	if _, err := sensorsRepo.Get("go-test"); err != nil {
		t.Fatalf("go-test sensor not found: %v", err)
	}

	server := setupTestServer()

	// 1. sensor.run - executa gofmt no proprio repo (rapido e deterministico)
	sensorParams, _ := json.Marshal(map[string]any{
		"name": "sensor.run",
		"arguments": map[string]any{
			"id":     "gofmt",
			"target": ".",
		},
	})
	sensorReq := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      20,
		Method:  "tools/call",
		Params:  sensorParams,
	}
	sensorResp := server.Handle(sensorReq)
	if sensorResp.Error != nil {
		t.Fatalf("sensor.run error: %v", sensorResp.Error)
	}

	// Verifica que o sensor retornou um resultado estruturado
	sensorResultJSON, _ := json.Marshal(sensorResp.Result)
	var sensorResult map[string]any
	_ = json.Unmarshal(sensorResultJSON, &sensorResult)
	if _, ok := sensorResult["passed"]; !ok {
		t.Fatal("sensor.run result missing 'passed' field")
	}

	// 2. judge.review - prepara review da rubric spec-adherence
	reviewParams, _ := json.Marshal(map[string]any{
		"name": "judge.review",
		"arguments": map[string]any{
			"rubric_id": "spec-adherence",
			"target":    "internal/sensors/adapters.go",
		},
	})
	reviewReq := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      21,
		Method:  "tools/call",
		Params:  reviewParams,
	}
	reviewResp := server.Handle(reviewReq)
	if reviewResp.Error != nil {
		t.Fatalf("judge.review error: %v", reviewResp.Error)
	}

	// Verifica que a review contem instructions e schema
	reviewResultJSON, _ := json.Marshal(reviewResp.Result)
	var reviewResult map[string]any
	_ = json.Unmarshal(reviewResultJSON, &reviewResult)
	if _, ok := reviewResult["instructions"]; !ok {
		t.Fatal("judge.review result missing 'instructions' field")
	}
	if _, ok := reviewResult["schema"]; !ok {
		t.Fatal("judge.review result missing 'schema' field")
	}

	// 3. contract.task.next - pega a proxima task da spec example-feature
	taskNextParams, _ := json.Marshal(map[string]any{
		"name": "contract.task.next",
		"arguments": map[string]any{
			"spec_id": "example-feature",
		},
	})
	taskNextReq := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      22,
		Method:  "tools/call",
		Params:  taskNextParams,
	}
	taskNextResp := server.Handle(taskNextReq)
	if taskNextResp.Error != nil {
		t.Fatalf("contract.task.next error: %v", taskNextResp.Error)
	}

	// Extrai o task_id da resposta
	taskNextJSON, _ := json.Marshal(taskNextResp.Result)
	t.Logf("task.next raw result: %s", string(taskNextJSON))
	var taskNextResult map[string]any
	_ = json.Unmarshal(taskNextJSON, &taskNextResult)
	taskMap, ok := taskNextResult["task"].(map[string]any)
	if !ok {
		t.Fatalf("contract.task.next result missing 'task' field. result=%v", taskNextResult)
	}
	taskID := taskMap["id"].(string)

	// 4. contract.task.complete - marca a task como completada com evidencias
	completeParams, _ := json.Marshal(map[string]any{
		"name": "contract.task.complete",
		"arguments": map[string]any{
			"task_id": taskID,
			"evidence": []map[string]any{
				{
					"kind":      "sensor_run",
					"sensor":    "gofmt",
					"passed":    sensorResult["passed"],
					"timestamp": "2026-05-01T12:00:00Z",
				},
				{
					"kind": "note",
					"text": "Review preparada via judge.review",
				},
			},
		},
	})
	completeReq := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      23,
		Method:  "tools/call",
		Params:  completeParams,
	}
	completeResp := server.Handle(completeReq)
	if completeResp.Error != nil {
		t.Fatalf("contract.task.complete error: %v", completeResp.Error)
	}

	// Verifica que a task foi marcada como completed
	completeJSON, _ := json.Marshal(completeResp.Result)
	var completeResult map[string]any
	_ = json.Unmarshal(completeJSON, &completeResult)
	completedTask, ok := completeResult["task"].(map[string]any)
	if !ok {
		t.Fatal("contract.task.complete result missing 'task' field")
	}
	if completedTask["status"] != "completed" {
		t.Fatalf("expected task status 'completed', got %v", completedTask["status"])
	}

	t.Logf("End-to-end flow completed successfully: sensor.run -> judge.review -> contract.task.next -> contract.task.complete")
}
