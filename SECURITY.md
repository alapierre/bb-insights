# Security Policy

## Supported Versions

Only the latest released version of `bb-insights` is supported. Please
upgrade to the latest release before reporting an issue.

## Reporting a Vulnerability

Please **do not** open a public GitHub issue for security vulnerabilities.

Instead, use GitHub's private vulnerability reporting: go to the
[Security tab](https://github.com/alapierre/bb-insights/security) of this
repository and select **Report a vulnerability**. This opens a private
advisory visible only to the maintainers, so the issue can be discussed and
fixed before it's disclosed publicly.

We aim to acknowledge new reports within a few days and to publish a fix or
mitigation as soon as reasonably possible, coordinating a disclosure
timeline with the reporter.

## Scope

`bb-insights` reads report files (JUnit XML, Go coverage profiles, SARIF)
and sends their content to the Bitbucket Cloud Code Insights API. Security
issues of particular interest include:

- Credential handling (Bitbucket tokens/app passwords) being logged,
  leaked, or sent to the wrong destination.
- Parser input handling (malformed or adversarial JUnit/coverage/SARIF
  files causing crashes, excessive resource use, or unexpected behavior).
- Supply-chain integrity of released binaries and container images (see the
  build provenance attestations published with each release).
