// Package model defines the internal representation shared by all report
// parsers and the Bitbucket Code Insights publisher. Parsers convert their
// input format into a Report; the publisher only ever deals with this type.
package model

// ReportType identifies the kind of report published to Bitbucket Code
// Insights. It maps directly to the Bitbucket "report_type" field.
type ReportType string

const (
	ReportTypeTest     ReportType = "TEST"
	ReportTypeCoverage ReportType = "COVERAGE"
	ReportTypeSecurity ReportType = "SECURITY"
)

// DefaultReporter identifies this application as the source of a report, for
// reports that don't set a more specific Reporter value themselves.
const DefaultReporter = "bb-insights"

// Result maps directly to the Bitbucket report "result" field.
type Result string

const (
	ResultPassed  Result = "PASSED"
	ResultFailed  Result = "FAILED"
	ResultPending Result = "PENDING"
)

// MetricType maps directly to the Bitbucket report data "type" field.
type MetricType string

const (
	MetricBoolean    MetricType = "BOOLEAN"
	MetricDate       MetricType = "DATE"
	MetricDuration   MetricType = "DURATION"
	MetricLink       MetricType = "LINK"
	MetricNumber     MetricType = "NUMBER"
	MetricPercentage MetricType = "PERCENTAGE"
	MetricText       MetricType = "TEXT"
)

// Metric is a single entry in the Bitbucket report "data" array. Value must
// be a type compatible with Type (bool, string, int64/float64, ...); the
// bitbucket package is responsible for encoding it correctly.
type Metric struct {
	Title string
	Type  MetricType
	Value any
}

// Report is the common internal representation produced by every parser and
// consumed by the publisher. ID must be deterministic for a given report
// kind so that republishing on the same commit updates the existing report
// instead of creating a duplicate.
type Report struct {
	ID          string
	Title       string
	Details     string
	Type        ReportType
	Result      Result
	Reporter    string
	Link        string
	Metrics     []Metric
	Annotations []Annotation
}
