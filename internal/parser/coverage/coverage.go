// Package coverage parses a Go coverage profile (coverage.out, produced by
// "go test -coverprofile=coverage.out") into the common internal report
// model.
package coverage

import (
	"fmt"
	"io"
	"math"

	"golang.org/x/tools/cover"

	"github.com/alapierre/bb-insights/internal/model"
)

// DefaultReportID is the deterministic Bitbucket report ID used for Go
// coverage reports unless overridden by the caller.
const DefaultReportID = "bb-insights-coverage"

type fileCoverage struct {
	fileName string
	covered  int
	total    int
}

// Parse reads a Go coverage profile and converts it into a model.Report
// with overall and per-file coverage aggregated into metrics. Per-file
// percentages are computed but not exposed as individual metrics: Bitbucket
// limits a report to at most 10 data entries, which can't accommodate an
// arbitrary number of source files. Future versions may turn low-coverage
// files into annotations instead (see CLAUDE.md).
func Parse(r io.Reader) (model.Report, error) {
	profiles, err := cover.ParseProfilesFromReader(r)
	if err != nil {
		return model.Report{}, fmt.Errorf("coverage: parsing coverage profile: %w", err)
	}

	files := make([]fileCoverage, 0, len(profiles))
	var totalStmts, coveredStmts, coveredFiles int
	for _, p := range profiles {
		fc := fileCoverage{fileName: p.FileName}
		for _, b := range p.Blocks {
			fc.total += b.NumStmt
			if b.Count > 0 {
				fc.covered += b.NumStmt
			}
		}
		if fc.covered > 0 {
			coveredFiles++
		}
		files = append(files, fc)
		totalStmts += fc.total
		coveredStmts += fc.covered
	}

	var overall float64
	if totalStmts > 0 {
		overall = float64(coveredStmts) / float64(totalStmts) * 100
	}
	overall = math.Round(overall*10) / 10

	report := model.Report{
		ID:      DefaultReportID,
		Title:   "Go Coverage Report",
		Details: fmt.Sprintf("%.1f%% overall coverage across %d files", overall, len(files)),
		Type:    model.ReportTypeCoverage,
		Result:  model.ResultPassed,
		Metrics: []model.Metric{
			{Title: "Coverage", Type: model.MetricPercentage, Value: overall},
			{Title: "Covered Statements", Type: model.MetricNumber, Value: coveredStmts},
			{Title: "Total Statements", Type: model.MetricNumber, Value: totalStmts},
			{Title: "Files", Type: model.MetricNumber, Value: len(files)},
			{Title: "Covered Files", Type: model.MetricNumber, Value: coveredFiles},
		},
	}
	return report, nil
}
