// Package junit parses the JUnit XML report produced by gotestsum
// (--junitfile) into the common internal report model.
package junit

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/alapierre/bb-insights/internal/model"
)

// DefaultReportID is the deterministic Bitbucket report ID used for Go unit
// test reports unless overridden by the caller.
const DefaultReportID = "bb-insights-tests"

// detailsTruncateLimit caps how much of a failure message/stack trace is
// forwarded to Bitbucket, to keep annotation payloads reasonably sized.
const detailsTruncateLimit = 2000

type testsuitesDoc struct {
	Suites []testsuite `xml:"testsuite"`
}

type testsuite struct {
	Name      string     `xml:"name,attr"`
	Time      string     `xml:"time,attr"`
	TestCases []testcase `xml:"testcase"`
}

type testcase struct {
	Classname string   `xml:"classname,attr"`
	Name      string   `xml:"name,attr"`
	Time      string   `xml:"time,attr"`
	Failure   *issue   `xml:"failure"`
	Error     *issue   `xml:"error"`
	Skipped   *skipped `xml:"skipped"`
}

type issue struct {
	Message string `xml:"message,attr"`
	Content string `xml:",chardata"`
}

type skipped struct {
	Message string `xml:"message,attr"`
}

// Parse reads a JUnit XML document (as produced by gotestsum) and converts
// it into a model.Report with test-count metrics and one annotation per
// failed test case.
func Parse(r io.Reader) (model.Report, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return model.Report{}, fmt.Errorf("junit: reading input: %w", err)
	}

	root, err := rootElementName(data)
	if err != nil {
		return model.Report{}, err
	}

	var suites []testsuite
	switch root {
	case "testsuites":
		var doc testsuitesDoc
		if err := xml.Unmarshal(data, &doc); err != nil {
			return model.Report{}, fmt.Errorf("junit: parsing <testsuites> document: %w", err)
		}
		suites = doc.Suites
	case "testsuite":
		var suite testsuite
		if err := xml.Unmarshal(data, &suite); err != nil {
			return model.Report{}, fmt.Errorf("junit: parsing <testsuite> document: %w", err)
		}
		suites = []testsuite{suite}
	default:
		return model.Report{}, fmt.Errorf("junit: unexpected root element <%s>, expected <testsuites> or <testsuite>", root)
	}

	return buildReport(suites), nil
}

func rootElementName(data []byte) (string, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", fmt.Errorf("junit: input is not valid XML: %w", err)
		}
		if start, ok := tok.(xml.StartElement); ok {
			return start.Name.Local, nil
		}
	}
}

func buildReport(suites []testsuite) model.Report {
	var total, passed, failed, skippedCount int
	var durationSeconds float64
	var annotations []model.Annotation

	for _, suite := range suites {
		for _, tc := range suite.TestCases {
			total++
			durationSeconds += parseSeconds(tc.Time)

			switch {
			case tc.Failure != nil || tc.Error != nil:
				failed++
				annotations = append(annotations, buildFailureAnnotation(tc))
			case tc.Skipped != nil:
				skippedCount++
			default:
				passed++
			}
		}
	}

	result := model.ResultPassed
	if failed > 0 {
		result = model.ResultFailed
	}

	report := model.Report{
		ID:      DefaultReportID,
		Title:   "Go Unit Test Results",
		Details: fmt.Sprintf("%d tests: %d passed, %d failed, %d skipped", total, passed, failed, skippedCount),
		Type:    model.ReportTypeTest,
		Result:  result,
		Metrics: []model.Metric{
			{Title: "Tests", Type: model.MetricNumber, Value: total},
			{Title: "Passed", Type: model.MetricNumber, Value: passed},
			{Title: "Failed", Type: model.MetricNumber, Value: failed},
			{Title: "Skipped", Type: model.MetricNumber, Value: skippedCount},
			{Title: "Duration", Type: model.MetricDuration, Value: int64(durationSeconds * 1000)},
		},
		Annotations: annotations,
	}
	return report
}

func buildFailureAnnotation(tc testcase) model.Annotation {
	failure := tc.Failure
	if failure == nil {
		failure = tc.Error
	}

	return model.Annotation{
		ExternalID: model.HashID("junit", tc.Classname, tc.Name),
		Type:       model.AnnotationBug,
		Severity:   model.SeverityHigh,
		Result:     model.AnnotationResultFailed,
		Summary:    fmt.Sprintf("%s.%s failed", tc.Classname, tc.Name),
		Details:    truncate(failureDetails(failure)),
	}
}

func failureDetails(failure *issue) string {
	message := strings.TrimSpace(failure.Message)
	content := strings.TrimSpace(failure.Content)

	switch {
	case content == "" || content == message:
		return message
	case message == "":
		return content
	default:
		return message + "\n\n" + content
	}
}

func truncate(s string) string {
	if len(s) <= detailsTruncateLimit {
		return s
	}
	return s[:detailsTruncateLimit] + "... (truncated)"
}

func parseSeconds(raw string) float64 {
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return v
}
