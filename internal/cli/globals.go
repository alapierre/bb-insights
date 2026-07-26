// Package cli wires the kong command-line interface: it resolves
// configuration, builds a bitbucket.Client and a publish.Publisher, and
// dispatches to each "publish" subcommand. It contains no report-parsing or
// HTTP logic of its own.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/alapierre/bb-insights/internal/bitbucket"
	"github.com/alapierre/bb-insights/internal/model"
	"github.com/alapierre/bb-insights/internal/publish"
)

// Globals holds the flags shared by every "publish" subcommand: where to
// publish (workspace/repo/commit), how to authenticate, and cross-cutting
// options like --dry-run.
type Globals struct {
	Workspace string `env:"BITBUCKET_WORKSPACE" required:"" help:"Bitbucket workspace (repository owner)."`
	Repo      string `env:"BITBUCKET_REPO_SLUG" required:"" help:"Bitbucket repository slug." name:"repo"`
	Commit    string `env:"BITBUCKET_COMMIT" required:"" help:"Commit hash to attach the report to."`

	// Exactly one of Token or the Username/AppPassword pair must be set; this
	// is validated by bitbucket.Auth.header() when the client is built,
	// rather than via kong's xor/and tags, which only model exclusivity
	// between individual flags and can't express "token XOR (user AND pass)".
	Token       string `env:"BB_INSIGHTS_TOKEN" help:"Bitbucket access token, sent as a Bearer token. Recommended for Bitbucket Pipelines."`
	Username    string `env:"BB_INSIGHTS_USERNAME" help:"Bitbucket username, for Basic Auth together with --app-password."`
	AppPassword string `env:"BB_INSIGHTS_APP_PASSWORD" help:"Bitbucket app password, for Basic Auth together with --username."`

	BaseURL string        `env:"BB_INSIGHTS_BASE_URL" default:"${defaultBaseURL}" help:"Bitbucket API base URL."`
	Timeout time.Duration `env:"BB_INSIGHTS_TIMEOUT" default:"30s" help:"HTTP request timeout."`

	Link     string `env:"BB_INSIGHTS_LINK" help:"URL linking back to the CI build, shown on the report."`
	ReportID string `env:"BB_INSIGHTS_REPORT_ID" help:"Override the default deterministic report ID."`

	DryRun bool `env:"BB_INSIGHTS_DRY_RUN" help:"Print the payload that would be sent instead of calling the Bitbucket API."`
}

func (g *Globals) auth() bitbucket.Auth {
	return bitbucket.Auth{Token: g.Token, Username: g.Username, AppPassword: g.AppPassword}
}

// publish finalizes report (applying --report-id if set) and either prints
// its Bitbucket payload (--dry-run) or publishes it for real.
func (g *Globals) publish(report model.Report) error {
	if g.ReportID != "" {
		report.ID = g.ReportID
	}

	if g.DryRun {
		return printDryRun(report, g.Link)
	}

	client, err := bitbucket.New(g.BaseURL, &http.Client{Timeout: g.Timeout}, g.auth(), nil)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), g.Timeout)
	defer cancel()

	return publish.New(client, g.Link).Publish(ctx, g.Workspace, g.Repo, g.Commit, report)
}

func printDryRun(report model.Report, link string) error {
	out := struct {
		Report      bitbucket.ReportPayload       `json:"report"`
		Annotations []bitbucket.AnnotationPayload `json:"annotations,omitempty"`
	}{
		Report:      publish.ToReportPayload(report, link),
		Annotations: publish.ToAnnotationPayloads(report.Annotations),
	}

	encoded, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("cli: encoding dry-run payload: %w", err)
	}
	if _, err := fmt.Fprintf(os.Stdout, "%s\n%s\n", report.ID, encoded); err != nil {
		return fmt.Errorf("cli: writing dry-run payload: %w", err)
	}
	return nil
}
