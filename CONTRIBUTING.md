# Contributing

Thanks for working on `deadcomments`.

## Development

Run the Go test suite:

```bash
go test ./...
```

Run the Docker Compose test stand:

```bash
docker compose up --build
```

Open:

- comments service: `http://localhost:8080`
- Docusaurus test stand: `http://localhost:3000/docs/intro`

Reset local Docker data:

```bash
docker compose down -v
```

## Pull Requests

Before opening a pull request:

- run `gofmt -w cmd internal`
- run `go test ./...`
- keep SQL inside repositories
- keep HTTP handlers thin
- keep Markdown, OAuth, moderation, identity, and event logic out of handlers
- update tests for behavior changes
- update README/AGENTS when architecture or workflows change

## Architecture

Read `AGENTS.md` before making non-trivial changes. It documents the project boundaries, event system, security rules, and testing expectations.

The event system is the preferred extension point for cross-cutting behavior such as audit logs, notifications, webhooks, and future async workflows.

