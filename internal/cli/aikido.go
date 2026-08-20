package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/alapierre/bb-insights/internal/model"
	"github.com/alapierre/bb-insights/internal/parser/aikido"
)

// AikidoCmd implements "publish aikido": a report from Aikido's local image
// scanner JSON output (not the SARIF export, which goes through "publish
// sarif" instead).
type AikidoCmd struct {
	Globals

	Input        string `required:"" type:"existingfile" name:"input" env:"BB_INSIGHTS_INPUT" help:"Path to the Aikido local image scan JSON report."`
	FailSeverity string `default:"high" enum:"critical,high,medium,low" name:"fail-severity" env:"BB_INSIGHTS_FAIL_SEVERITY" help:"Minimum finding severity that marks the report as FAILED (critical, high, medium, low)."`
	ExitCode     int    `default:"0" name:"exit-code" env:"BB_INSIGHTS_EXIT_CODE" help:"Exit with this code when findings exceed the fail-severity threshold (quality gate). Zero disables this behaviour and always exits successfully after publishing."`
}

func (c *AikidoCmd) Run() error {
	f, err := os.Open(c.Input)
	if err != nil {
		return fmt.Errorf("cli: opening Aikido report %q: %w", c.Input, err)
	}
	defer func() { _ = f.Close() }()

	threshold := model.Severity(strings.ToUpper(c.FailSeverity))
	report, err := aikido.Parse(f, threshold)
	if err != nil {
		return err
	}

	if err := c.publish(report); err != nil {
		return err
	}

	return qualityGateError(report, c.ExitCode, threshold)
}
