## What does this change do, and why?

## How was this tested?

<!-- go test ./... output, or manual verification steps. -->

## Checklist

- [ ] `go vet ./...`, `go test ./...` and `go build ./...` pass locally.
- [ ] Tests were added or updated for this change.
- [ ] Exported types/functions have doc comments.
- [ ] `README.md` was updated if user-facing behavior changed (new flags,
      subcommands, or env vars).
- [ ] New subcommand flags have an environment variable fallback (required
      for the Bitbucket Pipe invocation mode).
