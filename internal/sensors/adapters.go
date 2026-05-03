package sensors

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/ronaldofjc/local-harness/internal/common"
)

// GoTestAdapter normaliza a saida de `go test -json ./...`.
// Preferencialmente o sensor deve usar `go test -json` para output estruturado.
type GoTestAdapter struct{}

func (a *GoTestAdapter) Normalize(stdout, stderr string, exitCode int, ctx RunContext) common.SensorOutput {
	// Tenta parsear como JSON lines (go test -json)
	lines := strings.Split(stdout, "\n")
	var passed, failed, skipped int
	var violations []common.Violation

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// Nao e JSON, trata como texto simples
			continue
		}

		action, _ := event["Action"].(string)
		switch action {
		case "pass":
			passed++
		case "fail":
			failed++
			// Tenta extrair detalhes do teste falho
			test, _ := event["Test"].(string)
			pkg, _ := event["Package"].(string)
			output, _ := event["Output"].(string)
			v := common.Violation{
				Severity:    common.SeverityError,
				What:        fmt.Sprintf("Test failed: %s", test),
				Why:         output,
				Remediation: fmt.Sprintf("Fix test %s in package %s", test, pkg),
				FilesAffected: []string{},
			}
			violations = append(violations, v)
		case "skip":
			skipped++
		}
	}

	// Se nao conseguiu parsear nenhum evento JSON, fallback para texto
	if passed == 0 && failed == 0 && skipped == 0 {
		// Verifica se ha FAIL no texto
		if strings.Contains(stdout, "FAIL") || strings.Contains(stderr, "FAIL") || exitCode != 0 {
			return common.SensorOutput{
				Tool:       ctx.SensorID,
				Regulation: ctx.Regulation,
				Passed:     false,
				Summary:    fmt.Sprintf("%s: tests failed (exit %d)", ctx.SensorID, exitCode),
				Violations: []common.Violation{{
					Severity:    common.SeverityError,
					What:        "go test reported failures",
					Why:         stderr,
					Remediation: "Run go test locally to inspect failures",
				}},
			}
		}
		return common.SensorOutput{
			Tool:       ctx.SensorID,
			Regulation: ctx.Regulation,
			Passed:     true,
			Summary:    fmt.Sprintf("%s: all tests passed", ctx.SensorID),
			Violations: []common.Violation{},
		}
	}

	summary := fmt.Sprintf("%s: %d passed, %d failed, %d skipped", ctx.SensorID, passed, failed, skipped)
	return common.SensorOutput{
		Tool:       ctx.SensorID,
		Regulation: ctx.Regulation,
		Passed:     failed == 0,
		Summary:    summary,
		Violations: violations,
	}
}

// StaticcheckAdapter normaliza a saida de staticcheck.
type StaticcheckAdapter struct{}

func (a *StaticcheckAdapter) Normalize(stdout, stderr string, exitCode int, ctx RunContext) common.SensorOutput {
	lines := strings.Split(stdout, "\n")
	var violations []common.Violation

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Formato: arquivo:linha:coluna: mensagem (SAxxxx)
		parts := strings.SplitN(line, ":", 4)
		if len(parts) < 4 {
			continue
		}
		file := parts[0]
		lineNum, _ := strconv.Atoi(parts[1])
		msg := strings.TrimSpace(parts[3])

		violations = append(violations, common.Violation{
			Severity:      common.SeverityWarning,
			What:          msg,
			Why:           msg,
			Remediation:   fmt.Sprintf("Fix issue at %s:%d", file, lineNum),
			FilesAffected: []string{file},
			LinesAffected: [][2]int{{lineNum, lineNum}},
		})
	}

	passed := exitCode == 0 && len(violations) == 0
	summary := fmt.Sprintf("%s: %d issue(s)", ctx.SensorID, len(violations))
	if passed {
		summary = fmt.Sprintf("%s: no issues found", ctx.SensorID)
	}

	return common.SensorOutput{
		Tool:       ctx.SensorID,
		Regulation: ctx.Regulation,
		Passed:     passed,
		Summary:    summary,
		Violations: violations,
	}
}

// GoVetAdapter normaliza a saida de go vet.
type GoVetAdapter struct{}

func (a *GoVetAdapter) Normalize(stdout, stderr string, exitCode int, ctx RunContext) common.SensorOutput {
	// go vet escreve em stderr
	output := stderr
	if output == "" {
		output = stdout
	}

	lines := strings.Split(output, "\n")
	var violations []common.Violation

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Formato: arquivo:linha: mensagem
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}
		file := parts[0]
		lineNum, _ := strconv.Atoi(parts[1])
		msg := strings.TrimSpace(parts[2])

		violations = append(violations, common.Violation{
			Severity:      common.SeverityWarning,
			What:          msg,
			Why:           msg,
			Remediation:   fmt.Sprintf("Fix vet issue at %s:%d", file, lineNum),
			FilesAffected: []string{file},
			LinesAffected: [][2]int{{lineNum, lineNum}},
		})
	}

	passed := exitCode == 0 && len(violations) == 0
	summary := fmt.Sprintf("%s: %d issue(s)", ctx.SensorID, len(violations))
	if passed {
		summary = fmt.Sprintf("%s: no issues found", ctx.SensorID)
	}

	return common.SensorOutput{
		Tool:       ctx.SensorID,
		Regulation: ctx.Regulation,
		Passed:     passed,
		Summary:    summary,
		Violations: violations,
	}
}

// GoFmtAdapter normaliza a saida de gofmt -l.
type GoFmtAdapter struct{}

func (a *GoFmtAdapter) Normalize(stdout, stderr string, exitCode int, ctx RunContext) common.SensorOutput {
	lines := strings.Split(stdout, "\n")
	var violations []common.Violation

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		violations = append(violations, common.Violation{
			Severity:      common.SeverityWarning,
			What:          fmt.Sprintf("File needs formatting: %s", line),
			Why:           "gofmt -l listed this file",
			Remediation:   fmt.Sprintf("Run gofmt -w %s", line),
			FilesAffected: []string{line},
		})
	}

	passed := len(violations) == 0
	summary := fmt.Sprintf("%s: %d file(s) need formatting", ctx.SensorID, len(violations))
	if passed {
		summary = fmt.Sprintf("%s: all files formatted", ctx.SensorID)
	}

	return common.SensorOutput{
		Tool:       ctx.SensorID,
		Regulation: ctx.Regulation,
		Passed:     passed,
		Summary:    summary,
		Violations: violations,
	}
}

// TaskHarnessAdapter roda task do harness (Taskfile.yml).
type TaskHarnessAdapter struct{}

func (a *TaskHarnessAdapter) Normalize(stdout, stderr string, exitCode int, ctx RunContext) common.SensorOutput {
	// Usa passthrough como base mas com parsing leve de erros
	output := stdout
	if stderr != "" {
		output += "\n" + stderr
	}

	passed := exitCode == 0
	summary := fmt.Sprintf("%s: completed (exit %d)", ctx.SensorID, exitCode)
	var violations []common.Violation

	if !passed {
		violations = append(violations, common.Violation{
			Severity:      common.SeverityError,
			What:          "Harness task failed",
			Why:           output,
			Remediation:   "Inspect task output and fix the underlying issue",
			FilesAffected: []string{},
		})
		summary = fmt.Sprintf("%s: failed (exit %d)", ctx.SensorID, exitCode)
	}

	return common.SensorOutput{
		Tool:       ctx.SensorID,
		Regulation: ctx.Regulation,
		Passed:     passed,
		Summary:    summary,
		Violations: violations,
	}
}

// GoBenchAdapter normaliza a saida de go test -bench=. -benchmem.
type GoBenchAdapter struct{}

func (a *GoBenchAdapter) Normalize(stdout, stderr string, exitCode int, ctx RunContext) common.SensorOutput {
	lines := strings.Split(stdout, "\n")
	var violations []common.Violation
	benchCount := 0
	okCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Linhas de benchmark: BenchmarkX-N  num  ns/op  B/op  allocs/op
		if strings.HasPrefix(line, "Benchmark") {
			benchCount++
			if strings.Contains(line, "ns/op") {
				okCount++
			}
			continue
		}
		// Linhas de FAIL
		if strings.HasPrefix(line, "--- FAIL") {
			violations = append(violations, common.Violation{
				Severity:    common.SeverityError,
				What:        fmt.Sprintf("Benchmark failed: %s", line),
				Why:         line,
				Remediation: "Investigate benchmark failure",
			})
		}
	}

	// Verifica stderr por erros
	stderrLines := strings.Split(stderr, "\n")
	for _, line := range stderrLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		violations = append(violations, common.Violation{
			Severity:    common.SeverityWarning,
			What:        fmt.Sprintf("Benchmark stderr: %s", line),
			Why:         line,
			Remediation: "Check benchmark output for warnings",
		})
	}

	passed := exitCode == 0 && len(violations) == 0
	summary := fmt.Sprintf("%s: %d benchmarks run, %d ok", ctx.SensorID, benchCount, okCount)
	if !passed {
		summary = fmt.Sprintf("%s: %d benchmarks, %d violations (exit %d)", ctx.SensorID, benchCount, len(violations), exitCode)
	}

	return common.SensorOutput{
		Tool:       ctx.SensorID,
		Regulation: ctx.Regulation,
		Passed:     passed,
		Summary:    summary,
		Violations: violations,
	}
}

// DepCruiserAdapter analisa dependencias Go para violacoes arquiteturais.
// Espera a saida combinada de go vet + go list -f para detectar padroes.
type DepCruiserAdapter struct{}

func (a *DepCruiserAdapter) Normalize(stdout, stderr string, exitCode int, ctx RunContext) common.SensorOutput {
	var violations []common.Violation
	allOutput := stdout
	if stderr != "" {
		allOutput += "\n" + stderr
	}

	// Separa secoes: go vet output e ARCH_CHECK
	parts := strings.Split(allOutput, "---ARCH---")
	vetOutput := parts[0]
	archOutput := ""
	if len(parts) > 1 {
		archOutput = parts[1]
	}

	// Parse go vet output (mesmo padrao do GoVetAdapter)
	vetLines := strings.Split(vetOutput, "\n")
	for _, line := range vetLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}
		file := parts[0]
		lineNum, _ := strconv.Atoi(parts[1])
		msg := strings.TrimSpace(parts[2])
		violations = append(violations, common.Violation{
			Severity:      common.SeverityWarning,
			What:          msg,
			Why:           msg,
			Remediation:   fmt.Sprintf("Fix vet issue at %s:%d", file, lineNum),
			FilesAffected: []string{file},
			LinesAffected: [][2]int{{lineNum, lineNum}},
		})
	}

	// Parse ARCH_CHECK: procura handler -> repository imports
	archLines := strings.Split(archOutput, "\n")
	for _, line := range archLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, "ARCH_VIOLATION:") {
			msg := strings.TrimPrefix(line, "ARCH_VIOLATION:")
			msg = strings.TrimSpace(msg)
			violations = append(violations, common.Violation{
				Severity: common.SeverityError,
				What:     fmt.Sprintf("Architecture layer violation: %s", msg),
				Why:      "Handler package should not import repository package directly. Use service layer.",
				Remediation: fmt.Sprintf(
					"Introduce a port (interface) in the handler's domain and inject it. The handler must go through the service layer: %s",
					msg,
				),
			})
		}
	}

	// Se ha stderr com erros que nao sao warnings
	for _, line := range strings.Split(stderr, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, "ARCH_VIOLATION") {
			continue
		}
		if !strings.Contains(line, ":") {
			violations = append(violations, common.Violation{
				Severity: common.SeverityWarning,
				What:     fmt.Sprintf("Dep check warning: %s", line),
				Why:      line,
			})
		}
	}

	passed := len(violations) == 0
	summary := fmt.Sprintf("%s: %d architecture/dependency issues", ctx.SensorID, len(violations))
	if passed {
		summary = fmt.Sprintf("%s: no architecture violations found", ctx.SensorID)
	}

	return common.SensorOutput{
		Tool:       ctx.SensorID,
		Regulation: ctx.Regulation,
		Passed:     passed,
		Summary:    summary,
		Violations: violations,
	}
}

// PassthroughAdapter repassa stdout cru sem normalizacao.
type PassthroughAdapter struct{}

func (a *PassthroughAdapter) Normalize(stdout, stderr string, exitCode int, ctx RunContext) common.SensorOutput {
	passed := exitCode == 0
	summary := stdout
	if summary == "" {
		summary = stderr
	}
	if summary == "" {
		summary = fmt.Sprintf("%s: completed (exit %d)", ctx.SensorID, exitCode)
	}

	var violations []common.Violation
	if !passed {
		violations = append(violations, common.Violation{
			Severity:      common.SeverityError,
			What:          "Command failed",
			Why:           stderr,
			Remediation:   "Inspect command output",
			FilesAffected: []string{},
		})
	}

	return common.SensorOutput{
		Tool:       ctx.SensorID,
		Regulation: ctx.Regulation,
		Passed:     passed,
		Summary:    summary,
		Violations: violations,
	}
}
