# Contributing to bb-insights

Thanks for considering a contribution. This document covers how to propose
changes, what the codebase expects, and how to get a pull request merged.

## Before you start

- For anything beyond a small fix, please open an issue first (use the
  [feature request](.github/ISSUE_TEMPLATE/feature_request.yml) or
  [new report format](.github/ISSUE_TEMPLATE/new_report_format.yml) template)
  so the approach can be discussed before you invest time in an
  implementation.
- Security vulnerabilities must **not** be reported as public issues - see
  [SECURITY.md](SECURITY.md) for private reporting instructions.
- Participation in this project is governed by the
  [Code of Conduct](CODE_OF_CONDUCT.md).

## Project scope

`bb-insights` only transforms reports already produced by other tools (Go
tests, coverage, Trivy, JaCoCo, SARIF) into Bitbucket Cloud Code Insights. It
never runs tests, generates coverage, or performs scanning itself. Keep this
boundary in mind when proposing new functionality - see
[CLAUDE.md](CLAUDE.md) for the full architecture and design principles.

The codebase is organized into three layers:

- **Parser** (`internal/parser/<format>`) - reads an external report format
  and converts it into the internal model. Each format is an independent
  package.
- **Internal model** (`internal/model`) - the format-agnostic contract
  (`Report`, `Metric`, `Annotation`, `Severity`, `Location`) between parsers
  and the publisher.
- **Publisher** (`internal/bitbucket`, `internal/publish`) - talks to the
  Bitbucket Cloud Code Insights REST API and knows nothing about
  coverage.out, JUnit XML or SARIF.

Adding a new report format should only ever mean adding a new parser
package and a new CLI subcommand (`internal/cli`) - existing parsers should
never need to change.

## Development setup

Requires Go (see `go.mod` for the minimum version).

```bash
go build ./...
go vet ./...
go test ./...
```

There is no dependency on a real Bitbucket Cloud account for tests: HTTP
interactions with the Code Insights API are tested against
`httptest.Server`, and you should follow the same pattern for new
publisher-related tests.

## Making a change

- Work in small, incremental commits - a bug fix doesn't need surrounding
  cleanup, and a new parser doesn't need speculative abstractions for
  formats that don't exist yet.
- Write tests together with the production code, not as a follow-up.
  Parsers should have tests covering conversion into the internal model;
  publisher changes should have tests covering the generated Bitbucket
  payload.
- Document exported types and functions.
- Every subcommand's flags need an environment variable fallback (see the
  existing subcommands in `internal/cli`), since the Bitbucket Pipe
  invocation mode configures everything through `BB_INSIGHTS_*` env vars
  and picks the subcommand from `BB_INSIGHTS_REPORT_TYPE`.
- Error messages should explain what failed, why, and how to fix it - avoid
  generic errors.
- Update `README.md` when you add or change user-facing behavior (new
  flags, new subcommands, new env vars).

## Commit messages

Use an imperative, capitalized summary line (e.g. `Add fail-severity
support for Trivy and SARIF commands`), matching the existing `git log`
history. Keep the body, if any, focused on *why* the change was made.

## Submitting a pull request

1. Fork the repository and create a branch from `master`.
2. Make your change, following the guidelines above.
3. Ensure `go vet ./...`, `go test ./...` and `go build ./...` all pass -
   the same checks CI runs.
4. Open a pull request describing the change and the motivation behind it.
   Link the issue it addresses, if any.

A maintainer will review the PR; be responsive to review comments so the
change can land while it's still fresh.

## AI-assisted contributions

This project's own maintainer uses AI assistance (code review and parts of
implementation) as part of the workflow - see [AI_USAGE.md](AI_USAGE.md).
If you use AI tools to help write your contribution, that's fine, but you
are responsible for understanding, testing, and being able to explain every
line you submit. Do not submit AI-generated changes you have not reviewed
and verified yourself.
