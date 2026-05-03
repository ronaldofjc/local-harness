package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ronaldofjc/local-harness/internal/contracts"
	"github.com/ronaldofjc/local-harness/internal/guides"
	harnessfs "github.com/ronaldofjc/local-harness/internal/harness/fs"
	"github.com/ronaldofjc/local-harness/internal/judges"
	"github.com/ronaldofjc/local-harness/internal/mcp"
	"github.com/ronaldofjc/local-harness/internal/sensors"
	"github.com/ronaldofjc/local-harness/internal/sessions"
	"github.com/ronaldofjc/local-harness/internal/steering"
	"github.com/ronaldofjc/local-harness/internal/workflows"
)

const version = "0.1.0"

func main() {
	root := harnessfs.HarnessRoot()
	if root == "" {
		fmt.Fprintf(os.Stderr,
			"ERRO: HARNESS_ROOT nao configurado.\n"+
				"\n"+
				"Defina a variavel de ambiente HARNESS_ROOT apontando para\n"+
				"o diretorio .harness do seu workspace.\n"+
				"\n"+
				"Exemplo no opencode.json:\n"+
				"  \"env\": {\"HARNESS_ROOT\": \"/caminho/do/workspace/.harness\"}\n"+
				"\n"+
				"Exemplo no .cursor/mcp.json:\n"+
				"  \"env\": {\"HARNESS_ROOT\": \"/caminho/do/workspace/.harness\"}\n"+
				"\n"+
				"Exemplo na shell:\n"+
				"  export HARNESS_ROOT=/caminho/do/workspace/.harness\n")
		os.Exit(1)
	}

	// Inicializa repositorios
	guidesRepo := guides.NewFileSystemRepository(root)
	if err := guidesRepo.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to load guides: %v\n", err)
		os.Exit(1)
	}
	_ = guidesRepo.StartWatcher()

	sensorsRepo := sensors.NewFileSystemRepository(root)
	if err := sensorsRepo.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to load sensors: %v\n", err)
		os.Exit(1)
	}
	_ = sensorsRepo.StartWatcher()

	runner := sensors.NewRunner()
	sensorsService := sensors.NewService(sensorsRepo, runner)

	// Judges
	judgesRepo := judges.NewFileSystemRepository(root)
	if err := judgesRepo.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to load judges: %v\n", err)
		os.Exit(1)
	}
	_ = judgesRepo.StartWatcher()
	judgesValidator := judges.NewValidator()
	judgesService := judges.NewService(judgesRepo, judgesValidator)

	// Contracts
	specRepo := contracts.NewFileSystemSpecRepository(root)
	if err := specRepo.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to load specs: %v\n", err)
		os.Exit(1)
	}
	_ = specRepo.StartWatcher()

	taskRepo := contracts.NewFileSystemTaskRepository(root)
	if err := taskRepo.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to load tasks: %v\n", err)
		os.Exit(1)
	}

	contractsService := contracts.NewService(specRepo, taskRepo, sensorsService)

	// Sessions
	sessionsStore := sessions.NewStore(root)
	sessionsService := sessions.NewService(sessionsStore)

	// Steering
	steeringLog := steering.NewLog(root)
	steeringService := steering.NewService(steeringLog)

	// Workflows
	workflowsRepo := workflows.NewFileSystemRepository(root)
	if err := workflowsRepo.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to load workflows: %v\n", err)
		os.Exit(1)
	}

	// Server MCP
	server := mcp.NewServer(version, guidesRepo, sensorsService, judgesService, contractsService, sessionsService, steeringService, workflowsRepo)
	reader := bufio.NewReader(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "read error: %v\n", err)
			os.Exit(1)
		}

		var req mcp.JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}

		resp := server.Handle(req)
		if err := encoder.Encode(resp); err != nil {
			fmt.Fprintf(os.Stderr, "encode error: %v\n", err)
			os.Exit(1)
		}
	}
}
