// Package publish maps the internal report model onto Bitbucket Code
// Insights payloads and publishes them through a bitbucket.Client. It knows
// nothing about coverage.out, JUnit XML or SARIF.
package publish

import (
	"context"
	"fmt"

	"github.com/alapierre/bb-insights/internal/bitbucket"
	"github.com/alapierre/bb-insights/internal/model"
)

// Publisher publishes model.Report values to Bitbucket Cloud Code
// Insights.
type Publisher struct {
	client *bitbucket.Client
	// link is used as the report's "link" field when a report doesn't set
	// its own (e.g. a Bitbucket Pipelines build URL).
	link string
}

// New creates a Publisher backed by client. link is applied to every
// published report that doesn't already set its own Link; pass "" if none
// is available.
func New(client *bitbucket.Client, link string) *Publisher {
	return &Publisher{client: client, link: link}
}

// Publish creates or updates report and, if it has any, its annotations on
// the given commit.
func (p *Publisher) Publish(ctx context.Context, workspace, repoSlug, commit string, report model.Report) error {
	if report.ID == "" {
		return fmt.Errorf("publish: report %q has no ID set", report.Title)
	}

	if err := p.client.PutReport(ctx, workspace, repoSlug, commit, report.ID, ToReportPayload(report, p.link)); err != nil {
		return err
	}

	if len(report.Annotations) == 0 {
		return nil
	}

	return p.client.PutAnnotations(ctx, workspace, repoSlug, commit, report.ID, ToAnnotationPayloads(report.Annotations))
}

// ToReportPayload maps a Report onto the Bitbucket report payload, applying
// defaultLink and model.DefaultReporter as fallbacks. Exported so callers
// (e.g. a --dry-run CLI flag) can inspect exactly what would be sent without
// duplicating the mapping logic.
func ToReportPayload(report model.Report, defaultLink string) bitbucket.ReportPayload {
	reporter := report.Reporter
	if reporter == "" {
		reporter = model.DefaultReporter
	}

	link := report.Link
	if link == "" {
		link = defaultLink
	}

	return bitbucket.ReportPayload{
		Title:      report.Title,
		Details:    report.Details,
		ReportType: string(report.Type),
		Result:     string(report.Result),
		Reporter:   reporter,
		Link:       link,
		Data:       toDataItems(report.Metrics),
	}
}

func toDataItems(metrics []model.Metric) []bitbucket.ReportDataItem {
	if len(metrics) == 0 {
		return nil
	}
	items := make([]bitbucket.ReportDataItem, len(metrics))
	for i, m := range metrics {
		items[i] = bitbucket.ReportDataItem{Title: m.Title, Type: string(m.Type), Value: m.Value}
	}
	return items
}

// ToAnnotationPayloads maps Annotations onto Bitbucket annotation payloads.
func ToAnnotationPayloads(annotations []model.Annotation) []bitbucket.AnnotationPayload {
	items := make([]bitbucket.AnnotationPayload, len(annotations))
	for i, a := range annotations {
		item := bitbucket.AnnotationPayload{
			ExternalID:     a.ExternalID,
			AnnotationType: string(a.Type),
			Summary:        a.Summary,
			Details:        a.Details,
			Severity:       string(a.Severity),
			Result:         string(a.Result),
			Link:           a.Link,
		}
		if a.Location != nil {
			item.Path = a.Location.Path
			item.Line = a.Location.Line
		}
		items[i] = item
	}
	return items
}
