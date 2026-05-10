# syntax=docker/dockerfile:1.7

FROM golang:1.26-alpine AS build

WORKDIR /src
ENV GOTOOLCHAIN=local

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/deadcomments ./cmd/server

FROM alpine:3.20 AS runtime
ARG VERSION=dev
ARG REVISION=unknown
ARG SOURCE=https://github.com/dead-guru/comments

LABEL org.opencontainers.image.title="deadcomments" \
      org.opencontainers.image.description="Self-hosted comments and annotations for static sites" \
      org.opencontainers.image.source="${SOURCE}" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${REVISION}"

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S deadcomments \
    && adduser -S -D -H -h /app -s /sbin/nologin -G deadcomments deadcomments \
    && mkdir -p /app /data \
    && chown -R deadcomments:deadcomments /app /data

WORKDIR /app

COPY --from=build /out/deadcomments /app/deadcomments

USER deadcomments

ENV PORT=8080 \
    BASE_URL=http://localhost:8080 \
    DATABASE_PATH=/data/deadcomments.db

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD /app/deadcomments healthcheck

ENTRYPOINT ["/app/deadcomments"]
