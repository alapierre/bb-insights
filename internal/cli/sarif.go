package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/alapierre/bb-insights/internal/model"
	"github.com/alapierre/bb-insights/internal/parser/sarif"
)

// SarifCmd implements "publish sarif": a generic SARIF 2.1.0 report from any
// producer (golangci-lint, Semgrep, CodeQL, ...). "publish trivy" remains a
// separate subcommand for backward compatibility, backed by the same
// parser with its own preset options.
//
// Its default report ID (sarif.DefaultSarifReportID) is shared by every
// invocation of this subcommand, so publishing more than one SARIF-based
// report on the same commit (e.g. this alongside "trivy", or two different
// tools both run through "sarif") requires the global --report-id flag to
// keep them from overwriting each other.
type SarifCmd struct {
	Globals

	Input        string `required:"" type:"existingfile" name:"input" env:"BB_INSIGHTS_INPUT" help:"Path to the SARIF report."`
	Title        string `default:"SARIF Report" env:"BB_INSIGHTS_TITLE" help:"Report title shown in Bitbucket, e.g. the name of the tool that produced the SARIF report."`
	FailSeverity string `default:"high" enum:"critical,high,medium,low" name:"fail-severity" env:"BB_INSIGHTS_FAIL_SEVERITY" help:"Minimum finding severity that marks the report as FAILED (critical, high, medium, low)."`
	ExitCode     int    `default:"0" name:"exit-code" env:"BB_INSIGHTS_EXIT_CODE" help:"Exit with this code when findings exceed the fail-severity threshold (quality gate). Zero disables this behaviour and always exits successfully after publishing."`
}

func (c *SarifCmd) Run() error {
	f, err := os.Open(c.Input)
	if err != nil {
		return fmt.Errorf("cli: opening SARIF report %q: %w", c.Input, err)
	}
	defer func() { _ = f.Close() }()

	threshold := model.Severity(strings.ToUpper(c.FailSeverity))
	report, err := sarif.Parse(f, sarif.Options{
		ReportID:       sarif.DefaultSarifReportID,
		Title:          c.Title,
		IssueNoun:      "issues",
		ReportType:     model.ReportTypeBug,
		AnnotationType: model.AnnotationCodeSmell,
		FailThreshold:  threshold,
	})
	if err != nil {
		return err
	}

	if err := c.publish(report); err != nil {
		return err
	}

	return qualityGateError(report, c.ExitCode, threshold)
}
