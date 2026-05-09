# syntax=docker/dockerfile:1.7

FROM golang:1.25-alpine AS build

WORKDIR /src

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/deadcomments ./cmd/server

FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates tzdata curl \
    && addgroup -S deadcomments \
    && adduser -S -D -H -h /app -s /sbin/nologin -G deadcomments deadcomments \
    && mkdir -p /app /data \
    && chown -R deadcomments:deadcomments /app /data

WORKDIR /app

COPY --from=build /out/deadcomments /app/deadcomments
COPY --from=build /src/migrations /app/migrations
COPY --from=build /src/internal/templates /app/internal/templates
COPY --from=build /src/internal/static /app/internal/static
COPY --from=build /src/internal/widget /app/internal/widget

USER deadcomments

ENV PORT=8080 \
    BASE_URL=http://localhost:8080 \
    DATABASE_PATH=/data/deadcomments.db

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD curl -fsS http://127.0.0.1:${PORT}/healthz || exit 1

ENTRYPOINT ["/app/deadcomments"]
