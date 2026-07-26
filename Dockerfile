# Built by GoReleaser: the bb-insights binary is produced separately and
# copied into this already-built build context, so this Dockerfile has no
# Go build stage of its own.
FROM gcr.io/distroless/static-debian12:nonroot

COPY linux/amd64/bb-insights /usr/local/bin/bb-insights

ENTRYPOINT ["/usr/local/bin/bb-insights"]
