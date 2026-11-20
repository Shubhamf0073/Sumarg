# syntax=docker/dockerfile:1.7
# Sumarg backend image: one image, four entry points (docs/02 A4, docs/06 §3).
#   /app/schoolfleet serve --mode=api      api container (school portal + mobile apps)
#   /app/schoolfleet serve --mode=worker   worker container (notifications, retention, partitions, invoices)
#   /app/schoolfleet serve --mode=ingest   ingest container (MQTT/TLS broker for bus units on :8883)
#   /app/migrate up                        one-off, before each deploy (as sf_owner)
# Final stage is distroless/static: no shell, no package manager, runs as a non-root uid set by compose.

FROM golang:1.27-bookworm AS build
WORKDIR /src
ENV CGO_ENABLED=0 GOFLAGS=-trimpath GOTOOLCHAIN=local
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-s -w" -o /out/schoolfleet ./cmd/schoolfleet \
 && go build -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/schoolfleet /out/migrate /app/
ENV TZ=UTC
LABEL com.previa.project=schoolfleet
EXPOSE 8000 8883
ENTRYPOINT ["/app/schoolfleet"]
CMD ["serve", "--mode=api"]
