package sensors

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ronaldofjc/local-harness/internal/common"
)

// Runner executa comandos de sensors e aplica adapters de normalizacao.
type Runner struct {
	adapters map[string]Adapter
	timeout  time.Duration
}

// Adapter normaliza a saida de um comando para SensorOutput.
type Adapter interface {
	Normalize(stdout, stderr string, exitCode int, ctx RunContext) common.SensorOutput
}

// RunContext fornece contexto para o adapter.
type RunContext struct {
	SensorID   string
	Regulation common.Regulation
	Target     string
}

// NewRunner cria um novo runner com adapters built-in.
func NewRunner() *Runner {
	r := &Runner{
		adapters: make(map[string]Adapter),
		timeout:  60 * time.Second,
	}
	r.registerBuiltins()
	return r
}

// Run executa um sensor e retorna o output normalizado.
func (r *Runner) Run(sensor *Sensor, target string) common.SensorOutput {
	ctx := RunContext{
		SensorID:   sensor.ID,
		Regulation: sensor.Regulation,
		Target:     target,
	}

	// Resolve target
	if target == "" {
		target = sensor.Defaults["target"]
	}
	if target == "" {
		target = "."
	}

	// Substitui placeholder {target} no comando
	cmdStr := strings.ReplaceAll(sensor.Command, "{target}", target)

	// Executa comando com timeout
	execCtx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "sh", "-c", cmdStr)
	cmd.Dir = "." // usa cwd do processo; pode ser customizado futuramente

	stdoutBytes, err := cmd.Output()
	var stderrBytes []byte
	var exitCode int

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderrBytes = exitErr.Stderr
			exitCode = exitErr.ExitCode()
		} else if execCtx.Err() == context.DeadlineExceeded {
			return common.SensorOutput{
				Tool:               sensor.ID,
				Regulation:         sensor.Regulation,
				Passed:             false,
				Summary:            fmt.Sprintf("%s: timed out after %v", sensor.ID, r.timeout),
				Inconclusive:       true,
				InconclusiveReason: fmt.Sprintf("Sensor command timed out after %v.", r.timeout),
				Violations:         []common.Violation{},
			}
		} else if os.IsNotExist(err) {
			return common.SensorOutput{
				Tool:               sensor.ID,
				Regulation:         sensor.Regulation,
				Passed:             false,
				Summary:            fmt.Sprintf("%s: command not found", sensor.ID),
				Inconclusive:       true,
				InconclusiveReason: fmt.Sprintf("Sensor command could not be executed (ENOENT): %v", err),
				Violations:         []common.Violation{},
			}
		}
	} else {
		exitCode = cmd.ProcessState.ExitCode()
	}

	stdout := string(stdoutBytes)
	stderr := string(stderrBytes)

	// Aplica adapter
	adapter, ok := r.adapters[sensor.Adapter]
	if !ok {
		adapter = &PassthroughAdapter{}
	}

	return adapter.Normalize(stdout, stderr, exitCode, ctx)
}

// RegisterAdapter registra um adapter customizado.
func (r *Runner) RegisterAdapter(name string, adapter Adapter) {
	r.adapters[name] = adapter
}

func (r *Runner) registerBuiltins() {
	r.adapters["go-test"] = &GoTestAdapter{}
	r.adapters["staticcheck"] = &StaticcheckAdapter{}
	r.adapters["govet"] = &GoVetAdapter{}
	r.adapters["gofmt"] = &GoFmtAdapter{}
	r.adapters["go-bench"] = &GoBenchAdapter{}
	r.adapters["dep-cruiser"] = &DepCruiserAdapter{}
	r.adapters["task-harness"] = &TaskHarnessAdapter{}
	r.adapters["passthrough"] = &PassthroughAdapter{}
}
