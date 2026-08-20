package aikido

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

func TestParseSample(t *testing.T) {
	f, err := os.Open("../../../testdata/aikido/aikido-sample.json")
	if err != nil {
		t.Fatalf("opening fixture: %v", err)
	}
	defer func() { _ = f.Close() }()

	report, err := Parse(f, DefaultFailThreshold)
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
		t.Errorf("Result = %q, want %q (report has Critical/High findings)", report.Result, model.ResultFailed)
	}
	if !strings.Contains(report.Details, "example.com/sample-app:latest") {
		t.Errorf("Details = %q, want it to mention the scanned image", report.Details)
	}

	wantMetrics := map[string]any{
		"Critical": 1,
		"High":     2,
		"Medium":   1,
		"Low":      1,
		"Total":    5,
	}
	for title, want := range wantMetrics {
		if got := metricValue(t, report, title); got != want {
			t.Errorf("metric %q = %v, want %v", title, got, want)
		}
	}

	if len(report.Annotations) != 5 {
		t.Fatalf("len(Annotations) = %d, want 5", len(report.Annotations))
	}

	seenIDs := map[string]bool{}
	for _, a := range report.Annotations {
		if a.Type != model.AnnotationVulnerability {
			t.Errorf("annotation %q Type = %q, want %q", a.Summary, a.Type, model.AnnotationVulnerability)
		}
		if a.ExternalID == "" {
			t.Errorf("annotation %q has an empty ExternalID", a.Summary)
		}
		if seenIDs[a.ExternalID] {
			t.Errorf("duplicate ExternalID %q for annotation %q", a.ExternalID, a.Summary)
		}
		seenIDs[a.ExternalID] = true
	}
}

func TestParseBuildsAnnotationDetails(t *testing.T) {
	const doc = `{
		"image_name": "example.com/app",
		"findings": [
			{
				"type": "open_source",
				"severity": "Critical",
				"package_name": "openssl",
				"installed_version": "3.5.4-r0",
				"fix_versions": ["3.5.5-r0"],
				"cve_id": "CVE-2025-15467",
				"file": "/lib/apk/db/installed",
				"description": "CVE-2025-15467 in openssl"
			}
		]
	}`

	report, err := Parse(strings.NewReader(doc), DefaultFailThreshold)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(report.Annotations) != 1 {
		t.Fatalf("len(Annotations) = %d, want 1", len(report.Annotations))
	}

	a := report.Annotations[0]
	if a.Severity != model.SeverityCritical {
		t.Errorf("Severity = %q, want %q", a.Severity, model.SeverityCritical)
	}
	if a.Summary != "CVE-2025-15467: openssl" {
		t.Errorf("Summary = %q, want %q", a.Summary, "CVE-2025-15467: openssl")
	}
	if a.Location == nil || a.Location.Path != "/lib/apk/db/installed" {
		t.Errorf("Location = %+v, want a location with the finding's file", a.Location)
	}
	if !strings.Contains(a.Details, "Installed version: 3.5.4-r0") {
		t.Errorf("Details = %q, want it to mention the installed version", a.Details)
	}
	if !strings.Contains(a.Details, "Fix version(s): 3.5.5-r0") {
		t.Errorf("Details = %q, want it to mention the fix version", a.Details)
	}
	if a.Link != "https://nvd.nist.gov/vuln/detail/CVE-2025-15467" {
		t.Errorf("Link = %q, want the NVD detail page for the CVE", a.Link)
	}
}

func TestParseSkipsLinkForNonCVEIdentifiers(t *testing.T) {
	const doc = `{
		"findings": [
			{"type": "secrets", "severity": "High", "description": "hardcoded credential found"}
		]
	}`

	report, err := Parse(strings.NewReader(doc), DefaultFailThreshold)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(report.Annotations) != 1 {
		t.Fatalf("len(Annotations) = %d, want 1", len(report.Annotations))
	}

	a := report.Annotations[0]
	if a.Severity != model.SeverityHigh {
		t.Errorf("Severity = %q, want %q", a.Severity, model.SeverityHigh)
	}
	if a.Summary != "hardcoded credential found" {
		t.Errorf("Summary = %q, want it to fall back to the description", a.Summary)
	}
	if a.Link != "" {
		t.Errorf("Link = %q, want empty (no cve_id to build an NVD link from)", a.Link)
	}
	if a.Location != nil {
		t.Errorf("Location = %+v, want nil (finding has no file)", a.Location)
	}
}

func TestParseTruncatesOversizedSummaryAndDetails(t *testing.T) {
	longDescription := strings.Repeat("x", 3000)
	doc := `{"findings": [{"type": "secrets", "severity": "High", "description": "` + longDescription + `"}]}`

	report, err := Parse(strings.NewReader(doc), DefaultFailThreshold)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(report.Annotations) != 1 {
		t.Fatalf("len(Annotations) = %d, want 1", len(report.Annotations))
	}

	a := report.Annotations[0]
	if len(a.Summary) != summaryTruncateLimit+len("... (truncated)") {
		t.Errorf("len(Summary) = %d, want a summary truncated to %d chars plus the marker", len(a.Summary), summaryTruncateLimit)
	}
	if !strings.HasSuffix(a.Summary, "... (truncated)") {
		t.Errorf("Summary = %q, want it to end with the truncation marker", a.Summary)
	}
	if len(a.Details) != detailsTruncateLimit+len("... (truncated)") {
		t.Errorf("len(Details) = %d, want details truncated to %d chars plus the marker", len(a.Details), detailsTruncateLimit)
	}
}

func TestParseResultRespectsFailThreshold(t *testing.T) {
	const doc = `{"findings": [{"type": "open_source", "severity": "Low", "package_name": "foo", "description": "low finding"}]}`

	report, err := Parse(strings.NewReader(doc), DefaultFailThreshold)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if report.Result != model.ResultPassed {
		t.Errorf("Result = %q, want %q for a LOW-only report under the default HIGH threshold", report.Result, model.ResultPassed)
	}

	report, err = Parse(strings.NewReader(doc), model.SeverityLow)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if report.Result != model.ResultFailed {
		t.Errorf("Result = %q, want %q for a LOW finding with failThreshold = LOW", report.Result, model.ResultFailed)
	}
}

func TestParseRejectsInvalidJSON(t *testing.T) {
	_, err := Parse(strings.NewReader("not json at all"), DefaultFailThreshold)
	if err == nil {
		t.Fatal("Parse() expected an error for invalid input, got nil")
	}
}

func TestParseEmptyFindings(t *testing.T) {
	report, err := Parse(strings.NewReader(`{"findings":[]}`), DefaultFailThreshold)
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
