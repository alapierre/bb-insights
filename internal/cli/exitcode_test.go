package cli

import (
	"errors"
	"testing"

	"github.com/alapierre/bb-insights/internal/model"
)

func TestQualityGateError(t *testing.T) {
	tests := []struct {
		name      string
		report    model.Report
		exitCode  int
		threshold model.Severity
		wantNil   bool
		wantCode  int
		wantMsg   string
	}{
		{
			name:      "exit code 0 always returns nil",
			report:    model.Report{Result: model.ResultFailed},
			exitCode:  0,
			threshold: model.SeverityHigh,
			wantNil:   true,
		},
		{
			name:      "passed report returns nil even with non-zero exit code",
			report:    model.Report{Result: model.ResultPassed},
			exitCode:  1,
			threshold: model.SeverityHigh,
			wantNil:   true,
		},
		{
			name: "failed report returns ExitCodeError with finding count",
			report: model.Report{
				Result: model.ResultFailed,
				Annotations: []model.Annotation{
					{Severity: model.SeverityHigh},
					{Severity: model.SeverityCritical},
					{Severity: model.SeverityMedium}, // below threshold
				},
			},
			exitCode:  1,
			threshold: model.SeverityHigh,
			wantCode:  1,
			wantMsg:   "quality gate failed: 2 finding(s) at or above HIGH severity found",
		},
		{
			name: "custom exit code is preserved",
			report: model.Report{
				Result: model.ResultFailed,
				Annotations: []model.Annotation{
					{Severity: model.SeverityCritical},
				},
			},
			exitCode:  2,
			threshold: model.SeverityCritical,
			wantCode:  2,
			wantMsg:   "quality gate failed: 1 finding(s) at or above CRITICAL severity found",
		},
		{
			name: "failed report with no annotations above threshold",
			report: model.Report{
				Result: model.ResultFailed,
				Annotations: []model.Annotation{
					{Severity: model.SeverityLow},
					{Severity: model.SeverityMedium},
				},
			},
			exitCode:  1,
			threshold: model.SeverityHigh,
			wantCode:  1,
			wantMsg:   "quality gate failed: 0 finding(s) at or above HIGH severity found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := qualityGateError(tt.report, tt.exitCode, tt.threshold)
			if tt.wantNil {
				if err != nil {
					t.Fatalf("expected nil, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			var exitErr *ExitCodeError
			if !errors.As(err, &exitErr) {
				t.Fatalf("expected *ExitCodeError, got %T: %v", err, err)
			}
			if exitErr.Code != tt.wantCode {
				t.Errorf("code: got %d, want %d", exitErr.Code, tt.wantCode)
			}
			if exitErr.Error() != tt.wantMsg {
				t.Errorf("message: got %q, want %q", exitErr.Error(), tt.wantMsg)
			}
		})
	}
}
