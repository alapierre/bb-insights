package bitbucket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPutAnnotationsChunksRequests(t *testing.T) {
	const total = maxAnnotationsPerRequest + 50 // forces two chunks: 100 + 50

	var requestSizes []int
	var gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		var chunk []AnnotationPayload
		if err := json.NewDecoder(r.Body).Decode(&chunk); err != nil {
			t.Errorf("decoding request body: %v", err)
		}
		requestSizes = append(requestSizes, len(chunk))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := New(server.URL, server.Client(), Auth{Token: "tok"}, nil)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	annotations := make([]AnnotationPayload, total)
	for i := range annotations {
		annotations[i] = AnnotationPayload{ExternalID: "a", AnnotationType: "BUG", Summary: "s"}
	}

	if err := client.PutAnnotations(context.Background(), "ws", "repo", "commit", "report-id", annotations); err != nil {
		t.Fatalf("PutAnnotations() error: %v", err)
	}

	wantPath := "/repositories/ws/repo/commit/commit/reports/report-id/annotations"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}

	if len(requestSizes) != 2 || requestSizes[0] != maxAnnotationsPerRequest || requestSizes[1] != 50 {
		t.Fatalf("request chunk sizes = %v, want [%d 50]", requestSizes, maxAnnotationsPerRequest)
	}
}

func TestPutAnnotationsTruncatesAtReportLimit(t *testing.T) {
	var totalReceived int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var chunk []AnnotationPayload
		if err := json.NewDecoder(r.Body).Decode(&chunk); err != nil {
			t.Errorf("decoding request body: %v", err)
		}
		totalReceived += len(chunk)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := New(server.URL, server.Client(), Auth{Token: "tok"}, nil)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	annotations := make([]AnnotationPayload, maxAnnotationsPerReport+10)
	for i := range annotations {
		annotations[i] = AnnotationPayload{ExternalID: "a", AnnotationType: "BUG", Summary: "s"}
	}

	if err := client.PutAnnotations(context.Background(), "ws", "repo", "commit", "report-id", annotations); err != nil {
		t.Fatalf("PutAnnotations() error: %v", err)
	}

	if totalReceived != maxAnnotationsPerReport {
		t.Fatalf("annotations sent = %d, want %d (truncated to the per-report limit)", totalReceived, maxAnnotationsPerReport)
	}
}

func TestPutAnnotationsNoopOnEmptyInput(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := New(server.URL, server.Client(), Auth{Token: "tok"}, nil)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	if err := client.PutAnnotations(context.Background(), "ws", "repo", "commit", "report-id", nil); err != nil {
		t.Fatalf("PutAnnotations() error: %v", err)
	}
	if called {
		t.Fatal("PutAnnotations() with no annotations should not call the API")
	}
}
