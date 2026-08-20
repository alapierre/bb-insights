// Package aikido parses the JSON report produced by Aikido's local image
// scanner (not to be confused with Aikido's SARIF export, which is already
// handled by the generic sarif parser) into the common internal report
// model.
package aikido

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/alapierre/bb-insights/internal/model"
)

// DefaultReportID is the deterministic Bitbucket report ID used for Aikido
// image scan reports unless overridden by the caller.
const DefaultReportID = "bb-insights-aikido"

// DefaultTitle is the report title shown in Bitbucket.
const DefaultTitle = "Aikido Security Report"

// DefaultFailThreshold is the severity used when no threshold is supplied to
// Parse: only HIGH and CRITICAL findings fail the report.
const DefaultFailThreshold = model.SeverityHigh

// summaryTruncateLimit and detailsTruncateLimit cap how much of a finding's
// description is forwarded to Bitbucket. Bitbucket Cloud enforces its own
// (undocumented, but much shorter for "summary" than "details") limits on
// these annotation fields; a raw Aikido description without a package name
// or CVE ID to build a short summary from (e.g. a future secrets/SAST
// finding type) could otherwise exceed them and make the whole annotations
// POST batch fail rather than just this one finding.
const (
	summaryTruncateLimit = 450
	detailsTruncateLimit = 2000
)

// document is the top-level shape of an Aikido local image scan report.
type document struct {
	ImageName string    `json:"image_name"`
	Findings  []finding `json:"findings"`
}

// finding is a single entry in the report's "findings" array. Aikido's local
// scanner currently only emits "open_source" (SCA) findings, but other scan
// types (secrets, SAST, IaC) are expected to share this envelope - severity
// and description are always present, the rest are populated best-effort
// depending on what the finding type actually has.
type finding struct {
	Type             string   `json:"type"`
	Severity         string   `json:"severity"`
	PackageName      string   `json:"package_name"`
	InstalledVersion string   `json:"installed_version"`
	FixVersions      []string `json:"fix_versions"`
	CVEID            string   `json:"cve_id"`
	File             string   `json:"file"`
	Description      string   `json:"description"`
}

// Parse reads an Aikido local image scan JSON report and converts it into a
// model.Report with severity-bucketed metrics and one annotation per
// finding. The report Result is FAILED only if at least one finding meets
// failThreshold; lower-severity findings are still reported but don't affect
// the result.
func Parse(r io.Reader, failThreshold model.Severity) (model.Report, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return model.Report{}, fmt.Errorf("aikido: reading input: %w", err)
	}

	var doc document
	if err := json.Unmarshal(data, &doc); err != nil {
		return model.Report{}, fmt.Errorf("aikido: parsing report: %w", err)
	}

	if failThreshold == "" {
		failThreshold = DefaultFailThreshold
	}

	counts := map[model.Severity]int{}
	seen := map[string]int{}
	annotations := make([]model.Annotation, 0, len(doc.Findings))

	for _, f := range doc.Findings {
		severity := normalizeSeverity(f.Severity)
		counts[severity]++
		annotations = append(annotations, buildAnnotation(f, severity, seen))
	}

	total := len(annotations)
	result := model.ResultPassed
	for severity, count := range counts {
		if count > 0 && severity.AtLeast(failThreshold) {
			result = model.ResultFailed
			break
		}
	}

	details := fmt.Sprintf("%d vulnerabilities found", total)
	if doc.ImageName != "" {
		details = fmt.Sprintf("%d vulnerabilities found in %s", total, doc.ImageName)
	}

	report := model.Report{
		ID:      DefaultReportID,
		Title:   DefaultTitle,
		Details: details,
		Type:    model.ReportTypeSecurity,
		Result:  result,
		Metrics: []model.Metric{
			{Title: "Critical", Type: model.MetricNumber, Value: counts[model.SeverityCritical]},
			{Title: "High", Type: model.MetricNumber, Value: counts[model.SeverityHigh]},
			{Title: "Medium", Type: model.MetricNumber, Value: counts[model.SeverityMedium]},
			{Title: "Low", Type: model.MetricNumber, Value: counts[model.SeverityLow]},
			{Title: "Total", Type: model.MetricNumber, Value: total},
		},
		Annotations: annotations,
	}
	return report, nil
}

// normalizeSeverity maps an Aikido severity string onto model.Severity,
// falling back to LOW for a finding type that doesn't set one of the four
// known values.
func normalizeSeverity(raw string) model.Severity {
	switch strings.ToUpper(raw) {
	case "CRITICAL":
		return model.SeverityCritical
	case "HIGH":
		return model.SeverityHigh
	case "MEDIUM":
		return model.SeverityMedium
	default:
		return model.SeverityLow
	}
}

func buildAnnotation(f finding, severity model.Severity, seen map[string]int) model.Annotation {
	key := strings.Join([]string{f.Type, f.PackageName, f.InstalledVersion, f.CVEID, f.File}, "|")
	occurrence := seen[key]
	seen[key] = occurrence + 1

	idParts := []string{"aikido", f.Type, f.PackageName, f.InstalledVersion, f.CVEID, f.File}
	if occurrence > 0 {
		idParts = append(idParts, fmt.Sprint(occurrence))
	}

	summary := f.CVEID
	if f.PackageName != "" {
		if summary != "" {
			summary = fmt.Sprintf("%s: %s", summary, f.PackageName)
		} else {
			summary = f.PackageName
		}
	}
	if summary == "" {
		summary = f.Description
	}
	if summary == "" {
		summary = "Aikido finding"
	}
	summary = truncate(summary, summaryTruncateLimit)

	var details strings.Builder
	details.WriteString(f.Description)
	if f.InstalledVersion != "" {
		fmt.Fprintf(&details, "\nInstalled version: %s", f.InstalledVersion)
	}
	if len(f.FixVersions) > 0 {
		fmt.Fprintf(&details, "\nFix version(s): %s", strings.Join(f.FixVersions, ", "))
	}

	var loc *model.Location
	if f.File != "" {
		loc = &model.Location{Path: f.File}
	}

	link := ""
	if strings.HasPrefix(strings.ToUpper(f.CVEID), "CVE-") {
		link = "https://nvd.nist.gov/vuln/detail/" + f.CVEID
	}

	return model.Annotation{
		ExternalID: model.HashID(idParts...),
		Type:       model.AnnotationVulnerability,
		Severity:   severity,
		Result:     model.AnnotationResultFailed,
		Summary:    summary,
		Details:    truncate(strings.TrimSpace(details.String()), detailsTruncateLimit),
		Location:   loc,
		Link:       link,
	}
}

func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "... (truncated)"
}
