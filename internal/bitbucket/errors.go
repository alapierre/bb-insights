package bitbucket

import (
	"fmt"
	"strings"
)

// APIError represents a non-2xx response from the Bitbucket Cloud API. The
// response body is included verbatim (Bitbucket returns a JSON error object
// describing what went wrong) but never contains request credentials.
type APIError struct {
	Method     string
	Path       string
	StatusCode int
	Body       []byte
}

func (e *APIError) Error() string {
	msg := strings.TrimSpace(string(e.Body))
	if msg == "" {
		return fmt.Sprintf("bitbucket: %s %s returned status %d", e.Method, e.Path, e.StatusCode)
	}
	return fmt.Sprintf("bitbucket: %s %s returned status %d: %s", e.Method, e.Path, e.StatusCode, msg)
}
