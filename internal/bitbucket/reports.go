package bitbucket

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// maxReportDataElements is a hard limit imposed by the Bitbucket Cloud API:
// a report's "data" array may contain at most 10 entries.
const maxReportDataElements = 10

// ReportPayload is the JSON body sent to the "create or update a report"
// endpoint. Title, Details and ReportType are the only mandatory fields.
type ReportPayload struct {
	Title      string           `json:"title"`
	Details    string           `json:"details"`
	ReportType string           `json:"report_type"`
	Result     string           `json:"result,omitempty"`
	Reporter   string           `json:"reporter,omitempty"`
	Link       string           `json:"link,omitempty"`
	Data       []ReportDataItem `json:"data,omitempty"`
}

// ReportDataItem is a single metric entry in ReportPayload.Data.
type ReportDataItem struct {
	Title string `json:"title"`
	Type  string `json:"type"`
	Value any    `json:"value"`
}

// PutReport creates or updates a Code Insights report for the given commit.
// reportID must be stable across calls for the same logical report so that
// re-publishing on the same commit updates it in place instead of creating a
// duplicate; Bitbucket documentation recommends prefixing custom IDs with
// the reporting system's name to avoid collisions with other tools.
func (c *Client) PutReport(ctx context.Context, workspace, repoSlug, commit, reportID string, report ReportPayload) error {
	if len(report.Data) > maxReportDataElements {
		return fmt.Errorf("bitbucket: report %q has %d data elements, Bitbucket allows at most %d",
			reportID, len(report.Data), maxReportDataElements)
	}

	path := fmt.Sprintf("/repositories/%s/%s/commit/%s/reports/%s",
		url.PathEscape(workspace), url.PathEscape(repoSlug), url.PathEscape(commit), url.PathEscape(reportID))

	if _, err := c.do(ctx, http.MethodPut, path, report); err != nil {
		return fmt.Errorf("bitbucket: publishing report %q for commit %s: %w", reportID, commit, err)
	}
	return nil
}
