package bitbucket

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAuthHeader(t *testing.T) {
	tests := []struct {
		name    string
		auth    Auth
		want    string
		wantErr bool
	}{
		{
			name: "bearer token",
			auth: Auth{Token: "secret-token"},
			want: "Bearer secret-token",
		},
		{
			name: "basic auth",
			auth: Auth{Username: "user", AppPassword: "pass"},
			want: "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass")),
		},
		{
			name:    "no credentials",
			auth:    Auth{},
			wantErr: true,
		},
		{
			name:    "username without app password",
			auth:    Auth{Username: "user"},
			wantErr: true,
		},
		{
			name:    "app password without username",
			auth:    Auth{AppPassword: "pass"},
			wantErr: true,
		},
		{
			name:    "token and basic auth together",
			auth:    Auth{Token: "t", Username: "user", AppPassword: "pass"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.auth.header()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("header() = %q, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("header() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("header() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewRejectsInvalidAuth(t *testing.T) {
	if _, err := New("", nil, Auth{}, nil); err == nil {
		t.Fatal("New() with no auth configured should fail")
	}
}

func TestClientSendsAuthorizationHeader(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := New(server.URL, server.Client(), Auth{Token: "abc123"}, nil)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	if err := client.PutReport(context.Background(), "ws", "repo", "commit123", "report-id", ReportPayload{
		Title: "t", Details: "d", ReportType: "TEST",
	}); err != nil {
		t.Fatalf("PutReport() error: %v", err)
	}

	if gotAuth != "Bearer abc123" {
		t.Fatalf("Authorization header = %q, want %q", gotAuth, "Bearer abc123")
	}
}

func TestClientMapsErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"access denied"}}`))
	}))
	defer server.Close()

	client, err := New(server.URL, server.Client(), Auth{Token: "abc123"}, nil)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	err = client.PutReport(context.Background(), "ws", "repo", "commit123", "report-id", ReportPayload{
		Title: "t", Details: "d", ReportType: "TEST",
	})
	if err == nil {
		t.Fatal("PutReport() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "403") || !strings.Contains(err.Error(), "access denied") {
		t.Fatalf("PutReport() error = %v, want it to mention status 403 and the response body", err)
	}
}

func TestClientRespectsContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := New(server.URL, server.Client(), Auth{Token: "abc123"}, nil)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	err = client.PutReport(ctx, "ws", "repo", "commit123", "report-id", ReportPayload{
		Title: "t", Details: "d", ReportType: "TEST",
	})
	if err == nil {
		t.Fatal("PutReport() expected a context deadline error, got nil")
	}
}
