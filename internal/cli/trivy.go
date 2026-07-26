package cli

import (
	"fmt"
	"os"

	"github.com/alapierre/bb-insights/internal/parser/sarif"
)

// TrivyCmd implements "publish trivy".
type TrivyCmd struct {
	Globals

	Input string `required:"" type:"existingfile" name:"input" env:"BB_INSIGHTS_INPUT" help:"Path to the Trivy SARIF report."`
}

func (c *TrivyCmd) Run() error {
	f, err := os.Open(c.Input)
	if err != nil {
		return fmt.Errorf("cli: opening SARIF report %q: %w", c.Input, err)
	}
	defer f.Close()

	report, err := sarif.Parse(f, sarif.TrivyOptions())
	if err != nil {
		return err
	}

	return c.publish(report)
}
