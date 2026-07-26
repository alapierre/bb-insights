// Package jacoco parses a JaCoCo XML coverage report (jacoco.xml, produced
// by the JaCoCo Maven/Gradle plugin) into the common internal report model.
package jacoco

import (
	"encoding/xml"
	"fmt"
	"io"
	"math"

	"github.com/alapierre/bb-insights/internal/model"
)

// DefaultReportID is the deterministic Bitbucket report ID used for JaCoCo
// coverage reports unless overridden by the caller.
const DefaultReportID = "bb-insights-jacoco"

type xmlReport struct {
	XMLName  xml.Name     `xml:"report"`
	Packages []xmlPackage `xml:"package"`
	Counters []xmlCounter `xml:"counter"`
}

type xmlPackage struct {
	Name string `xml:"name,attr"`
}

type xmlCounter struct {
	Type    string `xml:"type,attr"`
	Missed  int    `xml:"missed,attr"`
	Covered int    `xml:"covered,attr"`
}

// Parse reads a JaCoCo XML report and converts it into a model.Report with
// line and branch coverage aggregated into metrics. Per-package or per-class
// detail isn't surfaced as individual metrics: Bitbucket limits a report to
// at most 10 data entries. Future versions may turn low-coverage files into
// annotations instead, matching the Go coverage report (see CLAUDE.md).
func Parse(r io.Reader) (model.Report, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return model.Report{}, fmt.Errorf("jacoco: reading report: %w", err)
	}

	var xr xmlReport
	if err := xml.Unmarshal(data, &xr); err != nil {
		return model.Report{}, fmt.Errorf("jacoco: parsing report: %w", err)
	}

	lineCovered, lineMissed, ok := counterFor(xr.Counters, "LINE")
	if !ok {
		return model.Report{}, fmt.Errorf("jacoco: report has no top-level LINE <counter> element; is this a JaCoCo XML report (not the HTML or CSV output)?")
	}
	branchCovered, branchMissed, _ := counterFor(xr.Counters, "BRANCH")

	lineCoverage := percentage(lineCovered, lineMissed)
	branchCoverage := percentage(branchCovered, branchMissed)

	report := model.Report{
		ID:      DefaultReportID,
		Title:   "JaCoCo Coverage Report",
		Details: fmt.Sprintf("%.1f%% line coverage, %.1f%% branch coverage across %d packages", lineCoverage, branchCoverage, len(xr.Packages)),
		Type:    model.ReportTypeCoverage,
		Result:  model.ResultPassed,
		Metrics: []model.Metric{
			{Title: "Coverage", Type: model.MetricPercentage, Value: lineCoverage},
			{Title: "Covered Lines", Type: model.MetricNumber, Value: lineCovered},
			{Title: "Total Lines", Type: model.MetricNumber, Value: lineCovered + lineMissed},
			{Title: "Branch Coverage", Type: model.MetricPercentage, Value: branchCoverage},
			{Title: "Covered Branches", Type: model.MetricNumber, Value: branchCovered},
			{Title: "Total Branches", Type: model.MetricNumber, Value: branchCovered + branchMissed},
			{Title: "Packages", Type: model.MetricNumber, Value: len(xr.Packages)},
		},
	}
	return report, nil
}

func counterFor(counters []xmlCounter, typ string) (covered, missed int, ok bool) {
	for _, c := range counters {
		if c.Type == typ {
			return c.Covered, c.Missed, true
		}
	}
	return 0, 0, false
}

func percentage(covered, missed int) float64 {
	total := covered + missed
	if total == 0 {
		return 0
	}
	pct := float64(covered) / float64(total) * 100
	return math.Round(pct*10) / 10
}
