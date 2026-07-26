package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/alapierre/bb-insights/internal/model"
	"github.com/alapierre/bb-insights/internal/parser/sarif"
)

// TrivyCmd implements "publish trivy".
type TrivyCmd struct {
	Globals

	Input        string `required:"" type:"existingfile" name:"input" env:"BB_INSIGHTS_INPUT" help:"Path to the Trivy SARIF report."`
	FailSeverity string `default:"high" enum:"critical,high,medium,low" name:"fail-severity" env:"BB_INSIGHTS_FAIL_SEVERITY" help:"Minimum vulnerability severity that marks the report as FAILED (critical, high, medium, low)."`
}

func (c *TrivyCmd) Run() error {
	f, err := os.Open(c.Input)
	if err != nil {
		return fmt.Errorf("cli: opening SARIF report %q: %w", c.Input, err)
	}
	defer func() { _ = f.Close() }()

	opts := sarif.TrivyOptions()
	opts.FailThreshold = model.Severity(strings.ToUpper(c.FailSeverity))

	report, err := sarif.Parse(f, opts)
	if err != nil {
		return err
	}

	return c.publish(report)
}
