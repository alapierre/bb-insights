package model

// AnnotationType maps directly to the Bitbucket annotation "annotation_type"
// field.
type AnnotationType string

const (
	AnnotationVulnerability AnnotationType = "VULNERABILITY"
	AnnotationCodeSmell     AnnotationType = "CODE_SMELL"
	AnnotationBug           AnnotationType = "BUG"
)

// AnnotationResult maps directly to the Bitbucket annotation "result" field.
type AnnotationResult string

const (
	AnnotationResultPassed  AnnotationResult = "PASSED"
	AnnotationResultFailed  AnnotationResult = "FAILED"
	AnnotationResultIgnored AnnotationResult = "IGNORED"
	AnnotationResultSkipped AnnotationResult = "SKIPPED"
)

// Severity maps directly to the Bitbucket annotation "severity" field.
type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityHigh     Severity = "HIGH"
	SeverityMedium   Severity = "MEDIUM"
	SeverityLow      Severity = "LOW"
)

var severityRank = map[Severity]int{
	SeverityLow:      1,
	SeverityMedium:   2,
	SeverityHigh:     3,
	SeverityCritical: 4,
}

// AtLeast reports whether s is at least as severe as threshold, e.g.
// SeverityCritical.AtLeast(SeverityHigh) is true. An unrecognized Severity
// ranks below every defined severity.
func (s Severity) AtLeast(threshold Severity) bool {
	return severityRank[s] >= severityRank[threshold]
}

// Location points at a file and, optionally, a line within it. A nil
// *Location on an Annotation means the finding could not be attributed to a
// specific file (e.g. a SARIF result without location information).
type Location struct {
	Path string
	Line int
}

// Annotation is a single finding attached to a Report. ExternalID must be
// deterministic for the same underlying finding across repeated runs on the
// same commit, so that republishing updates rather than duplicates it.
type Annotation struct {
	ExternalID string
	Type       AnnotationType
	Severity   Severity
	Result     AnnotationResult
	Summary    string
	Details    string
	Location   *Location
	Link       string
}
