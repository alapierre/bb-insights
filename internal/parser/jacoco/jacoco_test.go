package jacoco

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

func TestParseSampleReport(t *testing.T) {
	f, err := os.Open("../../../testdata/jacoco/sample.xml")
	if err != nil {
		t.Fatalf("opening fixture: %v", err)
	}
	defer f.Close()

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
		"Coverage":         90.0,
		"Covered Lines":    90,
		"Total Lines":      100,
		"Branch Coverage":  80.0,
		"Covered Branches": 16,
		"Total Branches":   20,
		"Packages":         2,
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

func TestParseRejectsInvalidXML(t *testing.T) {
	_, err := Parse(strings.NewReader("not a jacoco report"))
	if err == nil {
		t.Fatal("Parse() expected an error for invalid XML, got nil")
	}
}

func TestParseRejectsWrongRootElement(t *testing.T) {
	_, err := Parse(strings.NewReader(`<foo/>`))
	if err == nil {
		t.Fatal("Parse() expected an error for a non-<report> root element, got nil")
	}
}

func TestParseReportWithoutBranches(t *testing.T) {
	const xml = `<report name="no-branches">
  <package name="com/example/foo"/>
  <counter type="INSTRUCTION" missed="0" covered="10"/>
  <counter type="LINE" missed="0" covered="5"/>
  <counter type="BRANCH" missed="0" covered="0"/>
</report>`

	report, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if got := metricValue(t, report, "Branch Coverage"); got != 0.0 {
		t.Errorf("Branch Coverage = %v, want 0 when there are no branches", got)
	}
}

func TestParseRejectsReportWithoutLineCounter(t *testing.T) {
	const xml = `<report name="no-line-counter">
  <counter type="INSTRUCTION" missed="0" covered="10"/>
</report>`

	_, err := Parse(strings.NewReader(xml))
	if err == nil {
		t.Fatal("Parse() expected an error when the report has no LINE counter, got nil")
	}
}
