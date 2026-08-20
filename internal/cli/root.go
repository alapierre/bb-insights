package cli

import "github.com/alecthomas/kong"

// CLI is the root kong command tree for bb-insights.
type CLI struct {
	Publish PublishCmd `cmd:"" help:"Publish a report to Bitbucket Cloud Code Insights."`

	Version kong.VersionFlag `help:"Print the version and exit."`
}

// PublishCmd groups the report-specific publish subcommands.
type PublishCmd struct {
	Tests    TestsCmd    `cmd:"" help:"Publish a Go unit test report (JUnit XML from gotestsum)."`
	Coverage CoverageCmd `cmd:"" help:"Publish a Go coverage report (coverage.out)."`
	Trivy    TrivyCmd    `cmd:"" help:"Publish a Trivy security report (SARIF)."`
	Sarif    SarifCmd    `cmd:"" help:"Publish a generic SARIF report (e.g. golangci-lint, Semgrep, CodeQL)."`
	JaCoCo   JaCoCoCmd   `cmd:"" name:"jacoco" help:"Publish a JaCoCo Java coverage report (XML)."`
	Aikido   AikidoCmd   `cmd:"" help:"Publish an Aikido local image scan report (JSON)."`
}
