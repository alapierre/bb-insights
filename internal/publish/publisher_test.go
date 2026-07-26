package publish

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alapierre/bb-insights/internal/bitbucket"
	"github.com/alapierre/bb-insights/internal/model"
)

func TestToReportPayloadAppliesDefaults(t *testing.T) {
	report := model.Report{
		ID:      "bb-insights-tests",
		Title:   "Go Unit Test Results",
		Details: "1 test: 1 passed",
		Type:    model.ReportTypeTest,
		Result:  model.ResultPassed,
		Metrics: []model.Metric{
			{Title: "Tests", Type: model.MetricNumber, Value: 1},
		},
	}

	payload := ToReportPayload(report, "https://pipelines.example/build/1")

	if payload.Reporter != model.DefaultReporter {
		t.Errorf("Reporter = %q, want default %q", payload.Reporter, model.DefaultReporter)
	}
	if payload.Link != "https://pipelines.example/build/1" {
		t.Errorf("Link = %q, want the default link since the report didn't set one", payload.Link)
	}
	if payload.ReportType != string(model.ReportTypeTest) {
		t.Errorf("ReportType = %q, want %q", payload.ReportType, model.ReportTypeTest)
	}
	if len(payload.Data) != 1 || payload.Data[0].Title != "Tests" {
		t.Errorf("Data = %+v, want a single Tests entry", payload.Data)
	}
}

func TestToReportPayloadKeepsExplicitReporterAndLink(t *testing.T) {
	report := model.Report{
		Title:    "t",
		Details:  "d",
		Type:     model.ReportTypeTest,
		Reporter: "custom-reporter",
		Link:     "https://custom.example/link",
	}

	payload := ToReportPayload(report, "https://pipelines.example/build/1")

	if payload.Reporter != "custom-reporter" {
		t.Errorf("Reporter = %q, want %q (explicit value should not be overridden)", payload.Reporter, "custom-reporter")
	}
	if payload.Link != "https://custom.example/link" {
		t.Errorf("Link = %q, want %q (explicit value should not be overridden)", payload.Link, "https://custom.example/link")
	}
}

func TestToAnnotationPayloadsMapsLocation(t *testing.T) {
	annotations := []model.Annotation{
		{
			ExternalID: "abc123",
			Type:       model.AnnotationVulnerability,
			Severity:   model.SeverityHigh,
			Result:     model.AnnotationResultFailed,
			Summary:    "CVE-2021-0001",
			Location:   &model.Location{Path: "go.mod", Line: 3},
		},
		{
			ExternalID: "def456",
			Type:       model.AnnotationBug,
			Summary:    "TestFoo failed",
		},
	}

	payloads := ToAnnotationPayloads(annotations)

	if payloads[0].Path != "go.mod" || payloads[0].Line != 3 {
		t.Errorf("payloads[0] path/line = %q/%d, want go.mod/3", payloads[0].Path, payloads[0].Line)
	}
	if payloads[1].Path != "" || payloads[1].Line != 0 {
		t.Errorf("payloads[1] path/line = %q/%d, want empty (no Location set)", payloads[1].Path, payloads[1].Line)
	}
}

func TestPublishSendsReportAndAnnotations(t *testing.T) {
	var requests []string
	var lastAnnotationsBody []bitbucket.AnnotationPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.Method == http.MethodPost {
			if err := json.NewDecoder(r.Body).Decode(&lastAnnotationsBody); err != nil {
				t.Errorf("decoding annotations body: %v", err)
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := bitbucket.New(server.URL, server.Client(), bitbucket.Auth{Token: "tok"}, nil)
	if err != nil {
		t.Fatalf("bitbucket.New() error: %v", err)
	}

	publisher := New(client, "")
	report := model.Report{
		ID:      "bb-insights-trivy",
		Title:   "Trivy Security Report",
		Details: "1 vulnerability found",
		Type:    model.ReportTypeSecurity,
		Result:  model.ResultFailed,
		Annotations: []model.Annotation{
			{ExternalID: "abc", Type: model.AnnotationVulnerability, Summary: "CVE-2021-0001"},
		},
	}

	if err := publisher.Publish(context.Background(), "ws", "repo", "commit123", report); err != nil {
		t.Fatalf("Publish() error: %v", err)
	}

	wantRequests := []string{
		"PUT /repositories/ws/repo/commit/commit123/reports/bb-insights-trivy",
		"POST /repositories/ws/repo/commit/commit123/reports/bb-insights-trivy/annotations",
	}
	if len(requests) != len(wantRequests) {
		t.Fatalf("requests = %v, want %v", requests, wantRequests)
	}
	for i, want := range wantRequests {
		if requests[i] != want {
			t.Errorf("requests[%d] = %q, want %q", i, requests[i], want)
		}
	}
	if len(lastAnnotationsBody) != 1 || lastAnnotationsBody[0].ExternalID != "abc" {
		t.Errorf("annotations body = %+v, want a single entry with ExternalID %q", lastAnnotationsBody, "abc")
	}
}

func TestPublishSkipsAnnotationsCallWhenNoneExist(t *testing.T) {
	var requests []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := bitbucket.New(server.URL, server.Client(), bitbucket.Auth{Token: "tok"}, nil)
	if err != nil {
		t.Fatalf("bitbucket.New() error: %v", err)
	}

	publisher := New(client, "")
	report := model.Report{ID: "bb-insights-coverage", Title: "t", Details: "d", Type: model.ReportTypeCoverage}

	if err := publisher.Publish(context.Background(), "ws", "repo", "commit123", report); err != nil {
		t.Fatalf("Publish() error: %v", err)
	}

	if len(requests) != 1 {
		t.Fatalf("requests = %v, want exactly one PUT and no annotations call", requests)
	}
}

func TestPublishRejectsReportWithoutID(t *testing.T) {
	client, err := bitbucket.New("http://example.invalid", http.DefaultClient, bitbucket.Auth{Token: "tok"}, nil)
	if err != nil {
		t.Fatalf("bitbucket.New() error: %v", err)
	}

	publisher := New(client, "")
	err = publisher.Publish(context.Background(), "ws", "repo", "commit123", model.Report{Title: "no id"})
	if err == nil {
		t.Fatal("Publish() expected an error for a report without an ID, got nil")
	}
}
