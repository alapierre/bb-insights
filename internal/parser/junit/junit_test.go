package junit

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
	f, err := os.Open("../../../testdata/junit/sample.xml")
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
	if report.Type != model.ReportTypeTest {
		t.Errorf("Type = %q, want %q", report.Type, model.ReportTypeTest)
	}
	if report.Result != model.ResultFailed {
		t.Errorf("Result = %q, want %q (one test failed)", report.Result, model.ResultFailed)
	}

	wantMetrics := map[string]any{
		"Tests":   3,
		"Passed":  1,
		"Failed":  1,
		"Skipped": 1,
	}
	for title, want := range wantMetrics {
		if got := metricValue(t, report, title); got != want {
			t.Errorf("metric %q = %v, want %v", title, got, want)
		}
	}
	if got := metricValue(t, report, "Duration"); got != int64(3) {
		t.Errorf("metric %q = %v, want %v", "Duration", got, int64(3))
	}

	if len(report.Annotations) != 1 {
		t.Fatalf("len(Annotations) = %d, want 1", len(report.Annotations))
	}
	ann := report.Annotations[0]
	if ann.Type != model.AnnotationBug {
		t.Errorf("annotation Type = %q, want %q", ann.Type, model.AnnotationBug)
	}
	if ann.Result != model.AnnotationResultFailed {
		t.Errorf("annotation Result = %q, want %q", ann.Result, model.AnnotationResultFailed)
	}
	if !strings.Contains(ann.Summary, "TestSubtract") {
		t.Errorf("annotation Summary = %q, want it to mention TestSubtract", ann.Summary)
	}
	if !strings.Contains(ann.Details, "expected 5, got 4") {
		t.Errorf("annotation Details = %q, want it to contain the failure message", ann.Details)
	}
	if ann.ExternalID == "" {
		t.Error("annotation ExternalID must not be empty")
	}
}

func TestParseIsDeterministic(t *testing.T) {
	read := func() model.Report {
		f, err := os.Open("../../../testdata/junit/sample.xml")
		if err != nil {
			t.Fatalf("opening fixture: %v", err)
		}
		defer func() { _ = f.Close() }()
		report, err := Parse(f)
		if err != nil {
			t.Fatalf("Parse() error: %v", err)
		}
		return report
	}

	a, b := read(), read()
	if a.Annotations[0].ExternalID != b.Annotations[0].ExternalID {
		t.Errorf("ExternalID is not deterministic across runs: %q != %q", a.Annotations[0].ExternalID, b.Annotations[0].ExternalID)
	}
}

func TestParseSingleSuiteWithoutWrapper(t *testing.T) {
	f, err := os.Open("../../../testdata/junit/sample_single_suite.xml")
	if err != nil {
		t.Fatalf("opening fixture: %v", err)
	}
	defer func() { _ = f.Close() }()

	report, err := Parse(f)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if got := metricValue(t, report, "Tests"); got != 1 {
		t.Errorf("Tests = %v, want 1", got)
	}
	if report.Result != model.ResultPassed {
		t.Errorf("Result = %q, want %q", report.Result, model.ResultPassed)
	}
}

func TestParseRejectsInvalidXML(t *testing.T) {
	_, err := Parse(strings.NewReader("not xml at all"))
	if err == nil {
		t.Fatal("Parse() expected an error for invalid input, got nil")
	}
}

func TestParseRejectsUnexpectedRoot(t *testing.T) {
	_, err := Parse(strings.NewReader(`<somethingelse></somethingelse>`))
	if err == nil {
		t.Fatal("Parse() expected an error for an unexpected root element, got nil")
	}
}
