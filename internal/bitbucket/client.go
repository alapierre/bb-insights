// Package bitbucket implements a client for the Bitbucket Cloud Code
// Insights REST API. See
// https://support.atlassian.com/bitbucket-cloud/docs/code-insights/ for the
// upstream documentation. This package knows nothing about coverage.out,
// JUnit XML or SARIF: it only sends already-built report and annotation
// payloads.
package bitbucket

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// DefaultBaseURL is the production Bitbucket Cloud API endpoint.
const DefaultBaseURL = "https://api.bitbucket.org/2.0"

// Auth selects how requests are authenticated against the Bitbucket Cloud
// API. Exactly one of Token or the Username/AppPassword pair must be set.
type Auth struct {
	// Token is a Bitbucket repository, project or workspace access token,
	// sent as an "Authorization: Bearer" header. This is the method
	// recommended by Atlassian and the natural fit for Bitbucket Pipelines.
	Token string

	// Username and AppPassword authenticate via HTTP Basic Auth, kept for
	// compatibility with pipelines that still rely on app passwords.
	Username    string
	AppPassword string
}

func (a Auth) header() (string, error) {
	hasToken := a.Token != ""
	hasUsername := a.Username != ""
	hasAppPassword := a.AppPassword != ""

	switch {
	case hasToken && (hasUsername || hasAppPassword):
		return "", fmt.Errorf("bitbucket: configure either a token or a username/app-password pair, not both")
	case hasToken:
		return "Bearer " + a.Token, nil
	case hasUsername && hasAppPassword:
		creds := base64.StdEncoding.EncodeToString([]byte(a.Username + ":" + a.AppPassword))
		return "Basic " + creds, nil
	case hasUsername || hasAppPassword:
		return "", fmt.Errorf("bitbucket: basic auth requires both a username and an app password")
	default:
		return "", fmt.Errorf("bitbucket: no authentication configured, set a token or a username/app-password pair")
	}
}

// Client publishes reports and annotations to Bitbucket Cloud Code
// Insights. A Client is safe for concurrent use.
type Client struct {
	baseURL    string
	httpClient *http.Client
	authHeader string
	logger     *slog.Logger
}

// New creates a Client. httpClient defaults to http.DefaultClient and
// logger to slog.Default() when nil. baseURL defaults to DefaultBaseURL when
// empty; tests can point it at an httptest.Server instead.
func New(baseURL string, httpClient *http.Client, auth Auth, logger *slog.Logger) (*Client, error) {
	header, err := auth.header()
	if err != nil {
		return nil, err
	}

	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &Client{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		httpClient: httpClient,
		authHeader: header,
		logger:     logger,
	}, nil
}

// do sends a request with the given JSON-encodable body (nil for none) and
// returns the raw response body on success. It never logs the Authorization
// header or credentials.
func (c *Client) do(ctx context.Context, method, path string, body any) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("bitbucket: encoding request body for %s %s: %w", method, path, err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("bitbucket: building request for %s %s: %w", method, path, err)
	}
	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bitbucket: request %s %s failed: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("bitbucket: reading response body for %s %s: %w", method, path, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{Method: method, Path: path, StatusCode: resp.StatusCode, Body: respBody}
	}

	return respBody, nil
}
