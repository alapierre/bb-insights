// Command bb-insights publishes Go test, Go coverage and Trivy security
// reports to Bitbucket Cloud Code Insights.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/alecthomas/kong"

	"github.com/alapierre/bb-insights/internal/bitbucket"
	"github.com/alapierre/bb-insights/internal/cli"
)

// version, commit and date are set via -ldflags at build time (see
// .goreleaser.yaml).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var app cli.CLI

	parser, err := kong.New(&app,
		kong.Name("bb-insights"),
		kong.Description("Publish Go test, coverage and Trivy security reports to Bitbucket Cloud Code Insights."),
		kong.UsageOnError(),
		kong.Vars{
			"defaultBaseURL": bitbucket.DefaultBaseURL,
			"version":        fmt.Sprintf("%s (commit %s, built %s)", version, commit, date),
		},
	)
	if err != nil {
		panic(err)
	}

	// When invoked with no arguments at all - e.g. as a Bitbucket Pipe
	// (`pipe: docker://...`), which preserves ENTRYPOINT and can only
	// configure the container via environment variables - fall back to
	// BB_INSIGHTS_REPORT_TYPE to pick the subcommand.
	args := os.Args[1:]
	if len(args) == 0 {
		pipeArgs, err := cli.PipeModeArgs()
		parser.FatalIfErrorf(err)
		if pipeArgs != nil {
			args = pipeArgs
		}
	}

	ctx, err := parser.Parse(args)
	parser.FatalIfErrorf(err)

	if err := ctx.Run(); err != nil {
		var exitErr *cli.ExitCodeError
		if errors.As(err, &exitErr) {
			fmt.Fprintln(os.Stderr, exitErr.Error())
			os.Exit(exitErr.Code)
		}
		ctx.FatalIfErrorf(err)
	}
}
