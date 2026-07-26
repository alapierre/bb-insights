// Package sarif parses a SARIF 2.1.0 report into the common internal report
// model. SARIF is produced by many tools (Trivy, golangci-lint, Semgrep,
// CodeQL, ...); Options lets each CLI subcommand customize how the parsed
// results are presented without duplicating the result-to-annotation
// conversion logic.
package sarif

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	sarif "github.com/owenrumney/go-sarif/v3/pkg/report/v210/sarif"

	"github.com/alapierre/bb-insights/internal/model"
)

// DefaultReportID is the deterministic Bitbucket report ID used for Trivy
// security reports unless overridden by the caller.
const DefaultReportID = "bb-insights-trivy"

// DefaultSarifReportID is the deterministic Bitbucket report ID used for
// generic SARIF reports unless overridden by the caller. Publishing more
// than one SARIF-based report on the same commit (e.g. this subcommand
// alongside "trivy", or two different tools both run through this
// subcommand) requires an explicit --report-id, since they'd otherwise
// share this default and overwrite each other.
const DefaultSarifReportID = "bb-insights-sarif"

// Options customizes the model.Report produced by Parse, so the same SARIF
// parsing logic can back multiple CLI subcommands.
type Options struct {
	// ReportID is the deterministic Bitbucket report ID.
	ReportID string
	// Title is the report title shown in Bitbucket.
	Title string
	// IssueNoun names what was counted, e.g. "vulnerabilities" or "issues",
	// used to build the report Details text.
	IssueNoun string
	// ReportType is the Bitbucket "report_type" for the report.
	ReportType model.ReportType
	// AnnotationType is the Bitbucket "annotation_type" applied to every
	// annotation built from a SARIF result.
	AnnotationType model.AnnotationType
	// FailThreshold is the minimum severity that marks the report Result as
	// FAILED. Findings below this severity are still counted in the metrics
	// and published as annotations, they just don't fail the report on their
	// own. Defaults to model.SeverityHigh when left zero-valued.
	FailThreshold model.Severity
}

// DefaultFailThreshold is the severity used when Options.FailThreshold is
// left unset: only HIGH and CRITICAL findings fail the report.
const DefaultFailThreshold = model.SeverityHigh

// TrivyOptions returns the Options used for "publish trivy", preserved
// exactly for backward compatibility.
func TrivyOptions() Options {
	return Options{
		ReportID:       DefaultReportID,
		Title:          "Trivy Security Report",
		IssueNoun:      "vulnerabilities",
		ReportType:     model.ReportTypeSecurity,
		AnnotationType: model.AnnotationVulnerability,
	}
}

// Parse reads a SARIF document and converts it into a model.Report with
// severity-bucketed metrics and one annotation per result. Results without
// location information still count towards the metrics but are published
// without a file/line attachment. The report Result is FAILED only if at
// least one finding meets opts.FailThreshold; lower-severity findings are
// still reported but don't affect the result.
func Parse(r io.Reader, opts Options) (model.Report, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return model.Report{}, fmt.Errorf("sarif: reading input: %w", err)
	}

	doc, err := sarif.FromBytes(data)
	if err != nil {
		return model.Report{}, fmt.Errorf("sarif: parsing SARIF document: %w", err)
	}

	counts := map[model.Severity]int{}
	seen := map[string]int{}
	var annotations []model.Annotation

	for _, run := range doc.Runs {
		rules := ruleIndex(run)
		for _, res := range run.Results {
			severity := resultSeverity(res, rules)
			counts[severity]++
			annotations = append(annotations, buildAnnotation(res, severity, opts.AnnotationType, seen))
		}
	}

	threshold := opts.FailThreshold
	if threshold == "" {
		threshold = DefaultFailThreshold
	}

	total := len(annotations)
	result := model.ResultPassed
	for severity, count := range counts {
		if count > 0 && severity.AtLeast(threshold) {
			result = model.ResultFailed
			break
		}
	}

	report := model.Report{
		ID:      opts.ReportID,
		Title:   opts.Title,
		Details: fmt.Sprintf("%d %s found", total, opts.IssueNoun),
		Type:    opts.ReportType,
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

func ruleIndex(run *sarif.Run) map[string]*sarif.ReportingDescriptor {
	rules := map[string]*sarif.ReportingDescriptor{}
	if run.Tool == nil || run.Tool.Driver == nil {
		return rules
	}
	for _, rule := range run.Tool.Driver.Rules {
		if rule.ID != nil {
			rules[*rule.ID] = rule
		}
	}
	return rules
}

// resultSeverity derives a CRITICAL/HIGH/MEDIUM/LOW severity for a result.
// SARIF's own "level" only distinguishes error/warning/note, which collapses
// CRITICAL into HIGH, so we prefer the rule's severity tags or its
// "security-severity" CVSS score (both of which Trivy sets) and only fall
// back to "level" when neither is present.
func resultSeverity(res *sarif.Result, rules map[string]*sarif.ReportingDescriptor) model.Severity {
	var rule *sarif.ReportingDescriptor
	if res.RuleID != nil {
		rule = rules[*res.RuleID]
	}

	if rule != nil && rule.Properties != nil {
		if sev, ok := severityFromTags(rule.Properties.Tags); ok {
			return sev
		}
		if score, ok := rule.Properties.Properties["security-severity"]; ok {
			if sev, ok := severityFromScore(score); ok {
				return sev
			}
		}
	}

	return severityFromLevel(res.Level)
}

func severityFromTags(tags []string) (model.Severity, bool) {
	for _, tag := range tags {
		switch strings.ToUpper(tag) {
		case "CRITICAL":
			return model.SeverityCritical, true
		case "HIGH":
			return model.SeverityHigh, true
		case "MEDIUM":
			return model.SeverityMedium, true
		case "LOW":
			return model.SeverityLow, true
		}
	}
	return "", false
}

func severityFromScore(raw any) (model.Severity, bool) {
	var score float64
	switch v := raw.(type) {
	case string:
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return "", false
		}
		score = parsed
	case float64:
		score = v
	default:
		return "", false
	}

	switch {
	case score >= 9.0:
		return model.SeverityCritical, true
	case score >= 7.0:
		return model.SeverityHigh, true
	case score >= 4.0:
		return model.SeverityMedium, true
	default:
		return model.SeverityLow, true
	}
}

func severityFromLevel(level string) model.Severity {
	switch level {
	case "error":
		return model.SeverityHigh
	case "warning":
		return model.SeverityMedium
	default:
		return model.SeverityLow
	}
}

func buildAnnotation(res *sarif.Result, severity model.Severity, annotationType model.AnnotationType, seen map[string]int) model.Annotation {
	ruleID := ""
	if res.RuleID != nil {
		ruleID = *res.RuleID
	}

	message := ""
	if res.Message != nil && res.Message.Text != nil {
		message = strings.TrimSpace(*res.Message.Text)
	}

	loc := resultLocation(res)

	key := ruleID
	if loc != nil {
		key = fmt.Sprintf("%s|%s|%d", ruleID, loc.Path, loc.Line)
	}
	occurrence := seen[key]
	seen[key] = occurrence + 1

	idParts := []string{"sarif", ruleID}
	if loc != nil {
		idParts = append(idParts, loc.Path, strconv.Itoa(loc.Line))
	}
	if occurrence > 0 {
		idParts = append(idParts, strconv.Itoa(occurrence))
	}

	summary := ruleID
	if pkg := extractPackage(message); pkg != "" {
		summary = fmt.Sprintf("%s: %s", ruleID, pkg)
	}
	if summary == "" {
		summary = "Vulnerability finding"
	}

	return model.Annotation{
		ExternalID: model.HashID(idParts...),
		Type:       annotationType,
		Severity:   severity,
		Result:     model.AnnotationResultFailed,
		Summary:    summary,
		Details:    message,
		Location:   loc,
	}
}

func resultLocation(res *sarif.Result) *model.Location {
	if len(res.Locations) == 0 {
		return nil
	}
	phys := res.Locations[0].PhysicalLocation
	if phys == nil || phys.ArtifactLocation == nil || phys.ArtifactLocation.URI == nil {
		return nil
	}

	loc := &model.Location{Path: *phys.ArtifactLocation.URI}
	if phys.Region != nil && phys.Region.StartLine != nil {
		loc.Line = *phys.Region.StartLine
	}
	return loc
}

// extractPackage pulls the affected package name out of a Trivy result
// message, which conventionally starts with a "Package: <name>" line.
func extractPackage(message string) string {
	for _, line := range strings.Split(message, "\n") {
		if pkg, ok := strings.CutPrefix(line, "Package: "); ok {
			return strings.TrimSpace(pkg)
		}
	}
	return ""
}
