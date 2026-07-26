package cli

import (
	"fmt"
	"os"
)

// pipeReportTypeEnv selects which "publish" subcommand to run when
// bb-insights is invoked with no arguments at all, which is how Bitbucket
// Pipelines runs a Docker image referenced as a pipe
// (`pipe: docker://...`): a pipe's ENTRYPOINT is preserved and receives no
// extra argv, only environment variables declared under the calling step's
// "variables:" block.
const pipeReportTypeEnv = "BB_INSIGHTS_REPORT_TYPE"

// PipeModeArgs returns the kong argv to use in place of an empty argument
// list, based on BB_INSIGHTS_REPORT_TYPE. It only ever selects the
// subcommand ("publish tests"/"coverage"/"trivy"/"sarif"/"jacoco"): the
// report file path and every other setting are resolved the normal way,
// from each flag's own env tag (e.g. --junit falls back to
// BB_INSIGHTS_JUNIT), so no argument building is needed beyond picking the
// subcommand.
//
// It returns (nil, nil) when BB_INSIGHTS_REPORT_TYPE isn't set, meaning
// normal argument parsing - and its usual "expected command" error for a
// truly empty invocation - should proceed unchanged.
func PipeModeArgs() ([]string, error) {
	reportType := os.Getenv(pipeReportTypeEnv)
	if reportType == "" {
		return nil, nil
	}

	switch reportType {
	case "tests":
		return []string{"publish", "tests"}, nil
	case "coverage":
		return []string{"publish", "coverage"}, nil
	case "trivy":
		return []string{"publish", "trivy"}, nil
	case "sarif":
		return []string{"publish", "sarif"}, nil
	case "jacoco":
		return []string{"publish", "jacoco"}, nil
	default:
		return nil, fmt.Errorf("cli: %s=%q is not a valid report type, expected one of: tests, coverage, trivy, sarif, jacoco",
			pipeReportTypeEnv, reportType)
	}
}
