package coverage

import (
	"os"
	"strings"
	"testing"

	"github.com/alapierre/bb-insights/internal/model"
)

func metricValue(t *testing.T, report model.Report, title string) any {
	t.Helper()
	for _, m := range report.Metrics {
		if m.Title == title {
			return m.Value
		}
	}
	t.Fatalf("metric %q not found in report", title)
	return nil
}

func TestParseSampleProfile(t *testing.T) {
	f, err := os.Open("../../../testdata/coverage/sample.out")
	if err != nil {
		t.Fatalf("opening fixture: %v", err)
	}
	defer func() { _ = f.Close() }()

	report, err := Parse(f)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if report.ID != DefaultReportID {
		t.Errorf("ID = %q, want %q", report.ID, DefaultReportID)
	}
	if report.Type != model.ReportTypeCoverage {
		t.Errorf("Type = %q, want %q", report.Type, model.ReportTypeCoverage)
	}
	if report.Result != model.ResultPassed {
		t.Errorf("Result = %q, want %q", report.Result, model.ResultPassed)
	}

	wantMetrics := map[string]any{
		"Coverage":           50.0,
		"Covered Statements": 2,
		"Total Statements":   4,
		"Files":              3,
		"Covered Files":      2,
	}
	for title, want := range wantMetrics {
		if got := metricValue(t, report, title); got != want {
			t.Errorf("metric %q = %v, want %v", title, got, want)
		}
	}

	if len(report.Metrics) > 10 {
		t.Fatalf("len(Metrics) = %d, Bitbucket allows at most 10 report data entries", len(report.Metrics))
	}
}

func TestParseRejectsInvalidProfile(t *testing.T) {
	_, err := Parse(strings.NewReader("not a coverage profile"))
	if err == nil {
		t.Fatal("Parse() expected an error for an invalid profile, got nil")
	}
}

func TestParseEmptyProfile(t *testing.T) {
	report, err := Parse(strings.NewReader("mode: set\n"))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if got := metricValue(t, report, "Coverage"); got != 0.0 {
		t.Errorf("Coverage = %v, want 0 for an empty profile", got)
	}
}
