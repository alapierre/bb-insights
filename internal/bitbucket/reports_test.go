package bitbucket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPutReportSendsExpectedRequest(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody ReportPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decoding request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := New(server.URL, server.Client(), Auth{Token: "tok"}, nil)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	report := ReportPayload{
		Title:      "Unit tests",
		Details:    "42 tests ran",
		ReportType: "TEST",
		Result:     "PASSED",
		Data: []ReportDataItem{
			{Title: "Tests", Type: "NUMBER", Value: float64(42)},
		},
	}

	err = client.PutReport(context.Background(), "myworkspace", "myrepo", "abcdef12", "bb-insights-tests", report)
	if err != nil {
		t.Fatalf("PutReport() error: %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodPut)
	}
	wantPath := "/repositories/myworkspace/myrepo/commit/abcdef12/reports/bb-insights-tests"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotBody.Title != report.Title || gotBody.ReportType != report.ReportType {
		t.Errorf("request body = %+v, want %+v", gotBody, report)
	}
}

func TestPutReportRejectsTooManyDataElements(t *testing.T) {
	client, err := New("http://example.invalid", http.DefaultClient, Auth{Token: "tok"}, nil)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	data := make([]ReportDataItem, maxReportDataElements+1)
	for i := range data {
		data[i] = ReportDataItem{Title: "x", Type: "NUMBER", Value: i}
	}

	err = client.PutReport(context.Background(), "ws", "repo", "commit", "report-id", ReportPayload{
		Title: "t", Details: "d", ReportType: "TEST", Data: data,
	})
	if err == nil {
		t.Fatal("PutReport() expected an error for too many data elements, got nil")
	}
}
