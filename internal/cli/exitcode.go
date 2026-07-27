package cli

import (
	"fmt"

	"github.com/alapierre/bb-insights/internal/model"
)

// ExitCodeError signals that a quality gate check failed and the process
// should exit with a specific non-zero code. It is distinct from ordinary
// errors (network failures, parse errors) so that main() can forward the
// caller-specified exit code without clobbering it with the generic exit-1
// error path.
type ExitCodeError struct {
	Code    int
	message string
}

func (e *ExitCodeError) Error() string {
	return e.message
}

// qualityGateError returns an *ExitCodeError when exitCode is non-zero and
// the report was marked FAILED (i.e. at least one finding reached the
// fail-severity threshold). Returns nil when exitCode is 0 or the report
// passed, keeping the default behaviour backwards compatible.
func qualityGateError(report model.Report, exitCode int, threshold model.Severity) error {
	if exitCode == 0 || report.Result != model.ResultFailed {
		return nil
	}
	count := 0
	for _, a := range report.Annotations {
		if a.Severity.AtLeast(threshold) {
			count++
		}
	}
	return &ExitCodeError{
		Code:    exitCode,
		message: fmt.Sprintf("quality gate failed: %d finding(s) at or above %s severity found", count, threshold),
	}
}
