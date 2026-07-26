package sarif

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

func annotationByRule(t *testing.T, report model.Report, ruleID string) model.Annotation {
	t.Helper()
	for _, a := range report.Annotations {
		if strings.HasPrefix(a.Summary, ruleID) {
			return a
		}
	}
	t.Fatalf("no annotation found for rule %q", ruleID)
	return model.Annotation{}
}

func TestParseTrivySample(t *testing.T) {
	f, err := os.Open("../../../testdata/sarif/trivy-sample.sarif.json")
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
	if report.Type != model.ReportTypeSecurity {
		t.Errorf("Type = %q, want %q", report.Type, model.ReportTypeSecurity)
	}
	if report.Result != model.ResultFailed {
		t.Errorf("Result = %q, want %q", report.Result, model.ResultFailed)
	}

	wantMetrics := map[string]any{
		"Critical": 1,
		"High":     0,
		"Medium":   1,
		"Low":      1,
		"Total":    3,
	}
	for title, want := range wantMetrics {
		if got := metricValue(t, report, title); got != want {
			t.Errorf("metric %q = %v, want %v", title, got, want)
		}
	}

	if len(report.Annotations) != 3 {
		t.Fatalf("len(Annotations) = %d, want 3", len(report.Annotations))
	}

	critical := annotationByRule(t, report, "CVE-2021-3711")
	if critical.Severity != model.SeverityCritical {
		t.Errorf("CVE-2021-3711 severity = %q, want %q (from rule tags)", critical.Severity, model.SeverityCritical)
	}
	if critical.Location == nil || critical.Location.Path != "usr/lib/x86_64-linux-gnu/libcrypto.so.1.1" {
		t.Errorf("CVE-2021-3711 location = %+v, want a location with the artifact URI", critical.Location)
	}
	if !strings.Contains(critical.Summary, "openssl") {
		t.Errorf("CVE-2021-3711 summary = %q, want it to mention the package name", critical.Summary)
	}

	medium := annotationByRule(t, report, "CVE-2022-9999")
	if medium.Severity != model.SeverityMedium {
		t.Errorf("CVE-2022-9999 severity = %q, want %q (from security-severity score)", medium.Severity, model.SeverityMedium)
	}

	low := annotationByRule(t, report, "CVE-2020-1111")
	if low.Severity != model.SeverityLow {
		t.Errorf("CVE-2020-1111 severity = %q, want %q (fallback from level)", low.Severity, model.SeverityLow)
	}
	if low.Location != nil {
		t.Errorf("CVE-2020-1111 location = %+v, want nil (no location in the SARIF result)", low.Location)
	}

	for _, a := range report.Annotations {
		if a.Type != model.AnnotationVulnerability {
			t.Errorf("annotation %q Type = %q, want %q", a.Summary, a.Type, model.AnnotationVulnerability)
		}
		if a.ExternalID == "" {
			t.Errorf("annotation %q has an empty ExternalID", a.Summary)
		}
	}
}

func TestParseRejectsInvalidSarif(t *testing.T) {
	_, err := Parse(strings.NewReader("not json at all"))
	if err == nil {
		t.Fatal("Parse() expected an error for invalid input, got nil")
	}
}

func TestParseEmptyRuns(t *testing.T) {
	report, err := Parse(strings.NewReader(`{"version":"2.1.0","runs":[]}`))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if report.Result != model.ResultPassed {
		t.Errorf("Result = %q, want %q for a report with no findings", report.Result, model.ResultPassed)
	}
	if len(report.Annotations) != 0 {
		t.Errorf("len(Annotations) = %d, want 0", len(report.Annotations))
	}
}

func TestParseAssignsDistinctIDsToDuplicateFindings(t *testing.T) {
	const doc = `{
		"version": "2.1.0",
		"runs": [{
			"tool": {"driver": {"name": "Trivy", "rules": []}},
			"results": [
				{"ruleId": "CVE-X", "level": "error", "message": {"text": "Package: foo"},
				 "locations": [{"physicalLocation": {"artifactLocation": {"uri": "same.go"}, "region": {"startLine": 1}}}]},
				{"ruleId": "CVE-X", "level": "error", "message": {"text": "Package: foo"},
				 "locations": [{"physicalLocation": {"artifactLocation": {"uri": "same.go"}, "region": {"startLine": 1}}}]}
			]
		}]
	}`

	report, err := Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(report.Annotations) != 2 {
		t.Fatalf("len(Annotations) = %d, want 2", len(report.Annotations))
	}
	if report.Annotations[0].ExternalID == report.Annotations[1].ExternalID {
		t.Errorf("expected distinct ExternalIDs for repeated findings at the same location, got %q twice",
			report.Annotations[0].ExternalID)
	}
}
