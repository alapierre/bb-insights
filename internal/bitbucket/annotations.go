package bitbucket

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// Limits imposed by the Bitbucket Cloud API on Code Insights annotations.
const (
	maxAnnotationsPerRequest = 100
	maxAnnotationsPerReport  = 1000
)

// AnnotationPayload is a single entry sent to the bulk "create or update
// annotations" endpoint. ExternalID, AnnotationType and Summary are the only
// mandatory fields.
type AnnotationPayload struct {
	ExternalID     string `json:"external_id"`
	AnnotationType string `json:"annotation_type"`
	Summary        string `json:"summary"`
	Details        string `json:"details,omitempty"`
	Severity       string `json:"severity,omitempty"`
	Result         string `json:"result,omitempty"`
	Path           string `json:"path,omitempty"`
	Line           int    `json:"line,omitempty"`
	Link           string `json:"link,omitempty"`
}

// PutAnnotations bulk-uploads annotations for a previously created report,
// splitting the request into chunks that respect Bitbucket's
// 100-annotations-per-request limit. If the total exceeds Bitbucket's
// 1000-annotations-per-report limit, the excess is dropped and a warning is
// logged rather than silently discarded.
func (c *Client) PutAnnotations(ctx context.Context, workspace, repoSlug, commit, reportID string, annotations []AnnotationPayload) error {
	if len(annotations) == 0 {
		return nil
	}

	if len(annotations) > maxAnnotationsPerReport {
		c.logger.Warn("dropping annotations exceeding Bitbucket's per-report limit",
			"report_id", reportID, "total", len(annotations), "limit", maxAnnotationsPerReport)
		annotations = annotations[:maxAnnotationsPerReport]
	}

	path := fmt.Sprintf("/repositories/%s/%s/commit/%s/reports/%s/annotations",
		url.PathEscape(workspace), url.PathEscape(repoSlug), url.PathEscape(commit), url.PathEscape(reportID))

	for start := 0; start < len(annotations); start += maxAnnotationsPerRequest {
		end := min(start+maxAnnotationsPerRequest, len(annotations))
		chunk := annotations[start:end]

		if _, err := c.do(ctx, http.MethodPost, path, chunk); err != nil {
			return fmt.Errorf("bitbucket: publishing annotations %d-%d for report %q on commit %s: %w",
				start, end, reportID, commit, err)
		}
	}
	return nil
}
