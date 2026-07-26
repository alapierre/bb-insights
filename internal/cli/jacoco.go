package cli

import (
	"fmt"
	"os"

	"github.com/alapierre/bb-insights/internal/parser/jacoco"
)

// JaCoCoCmd implements "publish jacoco".
type JaCoCoCmd struct {
	Globals

	Input string `required:"" type:"existingfile" name:"input" env:"BB_INSIGHTS_INPUT" help:"Path to the JaCoCo XML coverage report."`
}

func (c *JaCoCoCmd) Run() error {
	f, err := os.Open(c.Input)
	if err != nil {
		return fmt.Errorf("cli: opening JaCoCo report %q: %w", c.Input, err)
	}
	defer func() { _ = f.Close() }()

	report, err := jacoco.Parse(f)
	if err != nil {
		return err
	}

	return c.publish(report)
}
