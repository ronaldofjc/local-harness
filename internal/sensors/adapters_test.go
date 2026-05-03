package sensors

import (
	"testing"

	"github.com/ronaldofjc/local-harness/internal/common"
)

func TestGoTestAdapter_Normalize(t *testing.T) {
	adapter := &GoTestAdapter{}
	ctx := RunContext{SensorID: "go-test", Regulation: common.RegulationMaintainability}

	tests := []struct {
		name     string
		stdout   string
		stderr   string
		exitCode int
		wantPass bool
	}{
		{
			name:     "all pass",
			stdout:   `{"Action":"pass","Test":"TestFoo","Package":"pkg"}`,
			exitCode: 0,
			wantPass: true,
		},
		{
			name:     "one fail",
			stdout:   `{"Action":"fail","Test":"TestBar","Package":"pkg"}`,
			exitCode: 1,
			wantPass: false,
		},
		{
			name:     "text fallback fail",
			stdout:   "FAIL\tpkg\t0.1s",
			exitCode: 1,
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapter.Normalize(tt.stdout, tt.stderr, tt.exitCode, ctx)
			if got.Passed != tt.wantPass {
				t.Errorf("got passed=%v, want %v", got.Passed, tt.wantPass)
			}
		})
	}
}

func TestGoFmtAdapter_Normalize(t *testing.T) {
	adapter := &GoFmtAdapter{}
	ctx := RunContext{SensorID: "gofmt", Regulation: common.RegulationMaintainability}

	tests := []struct {
		name     string
		stdout   string
		exitCode int
		wantPass bool
		wantLen  int
	}{
		{
			name:     "all formatted",
			stdout:   "",
			exitCode: 0,
			wantPass: true,
			wantLen:  0,
		},
		{
			name:     "needs formatting",
			stdout:   "main.go\nfoo/bar.go",
			exitCode: 0,
			wantPass: false,
			wantLen:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapter.Normalize(tt.stdout, "", tt.exitCode, ctx)
			if got.Passed != tt.wantPass {
				t.Errorf("got passed=%v, want %v", got.Passed, tt.wantPass)
			}
			if len(got.Violations) != tt.wantLen {
				t.Errorf("got %d violations, want %d", len(got.Violations), tt.wantLen)
			}
		})
	}
}

func TestStaticcheckAdapter_Normalize(t *testing.T) {
	adapter := &StaticcheckAdapter{}
	ctx := RunContext{SensorID: "staticcheck", Regulation: common.RegulationMaintainability}

	stdout := "main.go:42:9: unused variable x (U1000)\n"
	got := adapter.Normalize(stdout, "", 1, ctx)

	if got.Passed {
		t.Error("expected passed=false")
	}
	if len(got.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(got.Violations))
	}
	if got.Violations[0].FilesAffected[0] != "main.go" {
		t.Errorf("expected file main.go, got %s", got.Violations[0].FilesAffected[0])
	}
}

func TestPassthroughAdapter_Normalize(t *testing.T) {
	adapter := &PassthroughAdapter{}
	ctx := RunContext{SensorID: "custom", Regulation: common.RegulationFitness}

	tests := []struct {
		name     string
		stdout   string
		exitCode int
		wantPass bool
	}{
		{
			name:     "pass",
			stdout:   "ok",
			exitCode: 0,
			wantPass: true,
		},
		{
			name:     "fail",
			stdout:   "error",
			exitCode: 1,
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapter.Normalize(tt.stdout, "", tt.exitCode, ctx)
			if got.Passed != tt.wantPass {
				t.Errorf("got passed=%v, want %v", got.Passed, tt.wantPass)
			}
		})
	}
}
