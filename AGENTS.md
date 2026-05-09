# AGENTS.md

Guidance for agents and contributors working on `deadcomments`.

## Product Direction

`deadcomments` is a self-hosted, production-grade comments system for blogs and static websites.

Core constraints:

- Go + SQLite.
- Server-rendered HTML.
- No SPA.
- No React/Vue/Svelte in the product app.
- Public commenters do not authenticate.
- Admins authenticate through GitHub OAuth.
- Public widget is iframe-based.
- Markdown must be rendered and sanitized server-side.
- The backend must stay extensible for reactions, ratings, subscriptions, notifications, and authenticated commenters later.

Do not turn this into a microservice architecture. Keep the product small, sharp, and production-oriented.

## Architecture Rules

Respect the existing layering:

- `domain`: data models, enums, domain-level concepts.
- `repository`: SQL ownership. SQL belongs here.
- `service`: business logic and orchestration.
- `http`: thin request/response glue.
- `markdown`: Markdown rendering and sanitization.
- `auth`: GitHub OAuth and session primitives.
- `event`: domain event infrastructure and subscribers.
- `templates`: server-rendered HTML.
- `widget`: public embed loader.

Handlers must not contain:

- SQL.
- Markdown rendering.
- OAuth internals.
- moderation decision logic.
- audit/event persistence logic.

Handlers should parse input, call services, and render/redirect/respond.

Services should call repositories and publish domain events after successful domain changes.

Repositories should own SQL and mapping between rows and domain structs.

## Event System

The event system is the main extension mechanism for behavior that should happen because of a domain fact.

Examples:

- notify an admin about a new comment.
- write audit history.
- send Discord/Telegram/email notifications.
- trigger subscription delivery.
- emit webhooks.
- fan out future reaction/rating analytics.
- retry failed external deliveries.

Current pieces:

- `internal/domain/event.go` defines event types and the `domain.Event` model.
- `internal/event/bus.go` provides the in-process durable publisher.
- `internal/repository/event_repository.go` stores events and delivery state.
- `migrations/002_events.sql` creates `events` and `event_deliveries`.
- `internal/event/audit_handler.go` listens to comment events and writes `moderation_events`.
- `internal/service/events.go` contains service helpers for publishing.

Important rule: do not add cross-cutting side effects directly to HTTP handlers or core business methods when an event listener is a better fit.

Good:

- `CommentService` publishes `comment.created`.
- `AuditHandler` listens and writes moderation history.
- A future `DiscordNotificationHandler` listens and sends admin alerts.

Bad:

- `comments_handlers.go` sends Discord messages.
- `comment_service.go` directly calls email, Discord, Telegram, and webhook clients.
- admin handlers manually create audit rows.

Events must stay useful and current. When adding or changing important domain behavior, ask:

- Should this publish a domain event?
- Is the event name stable and meaningful?
- Does the payload include enough context for future listeners without exposing secrets?
- Should the event reference `site_id`, `page_id`, `comment_id`, or `actor_admin_id`?
- Does the event avoid raw IP addresses, raw user agents, OAuth tokens, and other sensitive data?
- Should a listener update audit history instead of direct service code?

Event names should describe facts in past tense or fact form:

- `comment.created`
- `comment.status_set`
- `comment.edited`
- `comment.ip_banned`
- `page.state_changed`
- `word_ban.created`
- `identity.created`
- `identity.secret_reset`

Keep event payloads compact. IDs belong in first-class event fields where possible. Payloads are for useful context such as status, reason, page key, lengths, or flags.

The current bus is synchronous and in-process, but events are durable. Future migration paths include:

- async dispatcher reading `events` and `event_deliveries`.
- Redis Streams consumer groups.
- Temporal workflows for retries, fan-out, delays, and long-running notification flows.

Business services should not need to change for that migration. Preserve the `event.Publisher` boundary.

## Security Rules

Never store raw IP addresses.

Never store raw user-agent strings.

Never expose GitHub OAuth tokens.

Never allow unsafe raw HTML from comments.

Always sanitize rendered Markdown.

Admin POST forms must use CSRF protection.

Admin sessions must remain HttpOnly and SameSite.

Public origin validation must remain enforced for comment creation.

Tripcode identity secrets must never be stored on comments or emitted in events.

If adding event payload fields, do not include:

- raw IP.
- raw user agent.
- raw email.
- OAuth tokens.
- session tokens.
- CSRF tokens.
- full request headers.
- submitted tripcode secrets.

Use existing hashes and IDs instead.

## Tripcode Identities

Tripcode identity is not authentication. It is a public identity hint for anonymous commenters.

Users may submit names as `Display Name##secret`. `IdentityService` must be the only place that parses and resolves that syntax.

Rules:

- plain names have no tripcode.
- anonymous tripcodes store only `tripcode_public`, never the submitted secret.
- reserved names must reject missing or invalid secrets.
- reserved identities attach `identity_id` and render badges.
- comments should use `author_display_name` for public display.
- do not put tripcode parsing or verification in HTTP handlers.
- do not put tripcode secrets in event payloads, logs, comments, or templates.

## Comments and Threading

The database supports arbitrary-depth threaded replies.

The public UI must not indent forever.

Rendering strategy:

- root comments render normally.
- replies render under the root thread.
- deeper replies are visually flattened under the root.
- deeper replies show `replying to @author`.
- mobile layout keeps indentation minimal.

Do not replace this with infinitely nested DOM indentation.

## Admin UI

The admin panel is server-rendered.

Keep it fast, dense, and operational. It should feel like a practical moderation tool, not a marketing site.

Use simple templates, manual CSS, and minimal vanilla JS.

Do not introduce a frontend framework.

## Database and Migrations

Add schema changes as new files in `migrations/`.

Do not edit already-applied migrations unless explicitly doing early pre-release cleanup and the user asks for it.

Migration filenames should sort in application order:

- `001_initial.sql`
- `002_events.sql`
- `003_some_feature.sql`

Keep migrations idempotent where practical with `IF NOT EXISTS`.

## Testing

Run:

```bash
go test ./...
```

Add or update focused tests when changing:

- Markdown sanitization.
- comment creation.
- moderation decisions.
- origin validation.
- page auto-creation.
- tree building/flattening.
- bans.
- event publishing.
- event handlers.

Tests should prove behavior, not implementation trivia.

## CI and Release Workflows

GitHub Actions workflows are part of the product surface. Keep them strict and meaningful.

Current workflows:

- `.github/workflows/ci.yml`: formatting, module consistency, vet, race tests, coverage, govulncheck, Docusaurus build, Docker Compose smoke test.
- `.github/workflows/codeql.yml`: CodeQL security analysis.
- `.github/workflows/release.yml`: tag-based release archives and GHCR image publishing.
- `.github/dependabot.yml`: dependency update automation.

Release archives must include runtime assets required by the binary:

- `migrations`
- `internal/templates`
- `internal/static`
- `internal/widget`

Do not add a CI job that is allowed to fail unless there is an explicit documented reason. CI should catch real regressions.

## Docusaurus Test Stand

The Docusaurus test stand lives in `examples/docusaurus`.

Use it to verify the public iframe widget visually and behaviorally:

```bash
DEADCOMMENTS_DEV_SEED=1 go run ./cmd/server
cd examples/docusaurus
npm install
npm start
```

The test stand is not the product frontend. It may use Docusaurus/React because Docusaurus does, but the comments product itself must remain server-rendered Go + vanilla JS.

## Cleanliness Expectations

Keep code boring and explicit.

Prefer small focused types and methods over clever abstractions.

Avoid duplicating business rules across handlers and services.

Do not add hidden global behavior.

Do not leave temporary databases, build outputs, `node_modules`, or smoke-test files in the repo.

Before finishing substantial work, check:

```bash
gofmt -w cmd internal
go test ./...
```

If the Docusaurus example changes, also run:

```bash
cd examples/docusaurus
npm install
npm run build
```

Then remove generated `node_modules`, `build`, and `.docusaurus` unless the user explicitly wants them kept.
