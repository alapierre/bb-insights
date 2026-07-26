package cli

import (
	"slices"
	"testing"
)

func TestPipeModeArgs(t *testing.T) {
	tests := []struct {
		name       string
		reportType string
		want       []string
		wantErr    bool
	}{
		{name: "unset", reportType: "", want: nil},
		{name: "tests", reportType: "tests", want: []string{"publish", "tests"}},
		{name: "coverage", reportType: "coverage", want: []string{"publish", "coverage"}},
		{name: "trivy", reportType: "trivy", want: []string{"publish", "trivy"}},
		{name: "sarif", reportType: "sarif", want: []string{"publish", "sarif"}},
		{name: "jacoco", reportType: "jacoco", want: []string{"publish", "jacoco"}},
		{name: "unknown", reportType: "bogus", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(pipeReportTypeEnv, tt.reportType)

			got, err := PipeModeArgs()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("PipeModeArgs() = %v, want an error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("PipeModeArgs() unexpected error: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("PipeModeArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}
