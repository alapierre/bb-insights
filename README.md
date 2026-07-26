# bb-insights

[![CI](https://github.com/alapierre/bb-insights/actions/workflows/ci.yml/badge.svg)](https://github.com/alapierre/bb-insights/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/alapierre/bb-insights)](LICENCE)

`bb-insights` publishes software quality and security reports to
[Bitbucket Cloud Code Insights](https://support.atlassian.com/bitbucket-cloud/docs/code-insights/).

Bitbucket Cloud has an excellent pull request experience, but unlike
competing platforms it lacks native integrations for many common report
formats such as Go coverage or Trivy SARIF. `bb-insights` bridges that
gap: it consumes reports already produced by other tools and publishes them
to Bitbucket in a form the Code Insights UI understands.

`bb-insights` does not run any analysis itself. Its job starts after
`gotestsum`, `go test -coverprofile` or `trivy` have already produced their
reports.

## Supported reports

| Report              | Source                                    | Produced by                          |
|---------------------|-------------------------------------------|--------------------------------------|
| Go unit tests       | `test-results/unit-tests.xml` (JUnit XML) | `gotestsum --junitfile ...`          |
| Go coverage         | `coverage.out`                            | `go test -coverprofile=coverage.out` |
| Trivy security scan | `testdata/sarif/trivy.sarif`              | `trivy image --format sarif ...`     |

## Installation

Download a prebuilt binary from the [releases page](https://github.com/alapierre/bb-insights/releases),
or build from source:

```bash
go install github.com/alapierre/bb-insights/cmd/bb-insights@latest
```

A Docker image is also published for use as a pipeline step:

```bash
docker run --rm -v "$PWD:/data" -w /data \
  alapierre/bb-insights:latest \
  publish tests --workspace myteam --repo myrepo --commit "$BITBUCKET_COMMIT" \
    --junit test-results/unit-tests.xml
```

## Usage

```bash
bb-insights publish tests \
  --workspace myteam --repo myrepo --commit "$BITBUCKET_COMMIT" \
  --junit test-results/unit-tests.xml

bb-insights publish coverage \
  --workspace myteam --repo myrepo --commit "$BITBUCKET_COMMIT" \
  --input coverage.out

bb-insights publish trivy \
  --workspace myteam --repo myrepo --commit "$BITBUCKET_COMMIT" \
  --input trivy.sarif
```

`--workspace`, `--repo` and `--commit` also fall back to the
`BITBUCKET_WORKSPACE`, `BITBUCKET_REPO_SLUG` and `BITBUCKET_COMMIT`
environment variables that Bitbucket Pipelines sets automatically, so in a
pipeline step you typically only need to pass the report path.

### Authentication

Exactly one of the following must be configured:

- `--token` (env `BB_INSIGHTS_TOKEN`): a Bitbucket repository, project or
  workspace [access token](https://support.atlassian.com/bitbucket-cloud/docs/access-tokens/),
  sent as a Bearer token. This is the recommended method for Bitbucket
  Pipelines.
- `--username` + `--app-password` (env `BB_INSIGHTS_USERNAME` /
  `BB_INSIGHTS_APP_PASSWORD`): HTTP Basic Auth, for compatibility with setups
  that still rely on app passwords.

### Other flags

| Flag          | Env                    | Description                                                       |
|---------------|------------------------|-------------------------------------------------------------------|
| `--base-url`  | `BB_INSIGHTS_BASE_URL` | Bitbucket API base URL (default `https://api.bitbucket.org/2.0`). |
| `--timeout`   | `BB_INSIGHTS_TIMEOUT`  | HTTP request timeout (default `30s`).                             |
| `--link`      | `BB_INSIGHTS_LINK`     | URL linking back to the CI build, shown on the report.            |
| `--report-id` | -                      | Override the default deterministic report ID.                     |
| `--dry-run`   | `BB_INSIGHTS_DRY_RUN`  | Print the JSON payload instead of calling the Bitbucket API.      |

Each subcommand's report path flag also has an env fallback: `--junit` reads
`BB_INSIGHTS_JUNIT`, and `--input` (on both `coverage` and `trivy`) reads
`BB_INSIGHTS_INPUT`.

Run `bb-insights publish <command> --help` for the full list.

## Bitbucket Pipelines example

```yaml
image: golang:1.26

pipelines:
  default:
    - step:
        name: Test, scan and report
        script:
          # BITBUCKET_WORKSPACE, BITBUCKET_REPO_SLUG and BITBUCKET_COMMIT are
          # injected by Bitbucket Pipelines automatically; bb-insights reads
          # them as fallbacks, so --workspace/--repo/--commit can be omitted
          # below. BB_INSIGHTS_TOKEN is NOT injected automatically: it must
          # be configured once as a secured repository variable (Repository
          # settings > Repository variables), see below.
          - go install gotest.tools/gotestsum@latest
          - gotestsum --format pkgname --junitfile test-results/unit-tests.xml -- -coverprofile=coverage.out ./...
          - curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin
          - trivy fs --format sarif --output trivy.sarif .
          - curl -sfL https://github.com/alapierre/bb-insights/releases/latest/download/bb-insights_linux_amd64 -o /usr/local/bin/bb-insights
          - chmod +x /usr/local/bin/bb-insights
          - bb-insights publish tests --junit test-results/unit-tests.xml
          - bb-insights publish coverage --input coverage.out
          - bb-insights publish trivy --input trivy.sarif
```

Set `BB_INSIGHTS_TOKEN` as a secured repository variable pointing at a
repository access token with the `repository:write` scope (this is the
scope Bitbucket Cloud requires for the Code Insights reports API; the
pipeline-oriented `pipeline:write` scope, which explicitly covers uploading
code insights, also works).

## Use as a Bitbucket Pipe

The same Docker image published above can also be referenced directly as a
[pipe](https://support.atlassian.com/bitbucket-cloud/docs/write-a-pipe-for-bitbucket-pipelines/)
(`pipe: docker://...`), instead of installing the binary in the script.
Unlike a step-level `image:` (which overrides the container's entrypoint so
your `script:` commands can run inside it), a `pipe:` preserves the image's
own `ENTRYPOINT` and only lets you configure it through environment
variables - there's no way to pass extra CLI flags. When bb-insights is
started with no arguments at all, it looks at `BB_INSIGHTS_REPORT_TYPE` to
decide which subcommand to run, then resolves every flag (`--junit`,
`--input`, `--token`, ...) from its usual env var, exactly as it would from
the command line:

```yaml
image: golang:1.26

pipelines:
  default:
    - step:
        name: Test, scan and report
        script:
          - go install gotest.tools/gotestsum@latest
          - gotestsum --format pkgname --junitfile test-results/unit-tests.xml -- -coverprofile=coverage.out ./...
          - curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin
          - trivy fs --format sarif --output trivy.sarif .

          - pipe: docker://lapierre/bb-insights:latest
            variables:
              BB_INSIGHTS_REPORT_TYPE: tests
              BB_INSIGHTS_JUNIT: test-results/unit-tests.xml
              BB_INSIGHTS_TOKEN: $BB_INSIGHTS_TOKEN

          - pipe: docker://lapierre/bb-insights:latest
            variables:
              BB_INSIGHTS_REPORT_TYPE: coverage
              BB_INSIGHTS_INPUT: coverage.out
              BB_INSIGHTS_TOKEN: $BB_INSIGHTS_TOKEN

          - pipe: docker://lapierre/bb-insights:latest
            variables:
              BB_INSIGHTS_REPORT_TYPE: trivy
              BB_INSIGHTS_INPUT: trivy.sarif
              BB_INSIGHTS_TOKEN: $BB_INSIGHTS_TOKEN
```

`BITBUCKET_WORKSPACE`, `BITBUCKET_REPO_SLUG` and `BITBUCKET_COMMIT` don't
need to be listed under `variables:` - Bitbucket injects its own default
variables into pipe containers automatically, same as any other step.
`BB_INSIGHTS_TOKEN` does need to be listed explicitly (as shown above):
only variables declared under a pipe's `variables:` are passed through from
repository/workspace variables into the container.

## Verifying a release

Every release publishes, alongside the binary and Docker image:

- a **SBOM** (Software Bill of Materials, SPDX format) for the binary,
  generated by [Syft](https://github.com/anchore/syft), and one embedded in
  the Docker image manifest via `docker buildx`'s native SBOM support;
- a **build provenance attestation** for both the binary and the Docker
  image, proving they were built by this repository's GitHub Actions
  workflow from a specific commit, not assembled or modified elsewhere.

Verify a downloaded binary with the [GitHub CLI](https://cli.github.com/):

```bash
gh attestation verify bb-insights_linux_amd64 --owner alapierre
```

Verify the Docker image:

```bash
gh attestation verify oci://index.docker.io/alapierre/bb-insights:latest --owner alapierre
```

## Design

The codebase separates three concerns, as required by the project's
architecture (see `CLAUDE.md`):

- **Parsers** (`internal/parser/{junit,coverage,sarif}`) read an external
  report format and convert it into the internal model. Adding a new report
  format means adding a new parser package; existing parsers are never
  touched.
- **Internal model** (`internal/model`) is the format-agnostic contract
  between parsers and the publisher: `Report`, `Metric`, `Annotation`,
  `Severity`, `Location`.
- **Publisher** (`internal/bitbucket`, `internal/publish`) knows how to talk
  to the Bitbucket Cloud Code Insights REST API and nothing about
  coverage.out, JUnit XML or SARIF.

Report and annotation IDs are deterministic (a fixed ID per report kind, and
a hash of stable identifying fields per annotation), so re-running a pipeline
step on the same commit updates the existing report instead of duplicating
it.

## Development

```bash
go build ./...
go vet ./...
go test ./...
```

No integration tests require a real Bitbucket Cloud account; HTTP
interactions are tested against `httptest.Server`.

## Security

See [SECURITY.md](SECURITY.md) for how to report a vulnerability.

## License

Apache License 2.0 - see [LICENCE](LICENCE) for the full text.
