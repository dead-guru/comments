# deadcomments

Self-hosted universal comments for blogs and static websites. Public commenters are anonymous. Admins sign in with GitHub OAuth. Comments render through a small iframe widget backed by Go, SQLite, server-rendered HTML, sanitized GitHub-flavored Markdown, and a layered backend.

## Run locally

```bash
go run ./cmd/server
```

Default local config:

```bash
BASE_URL=http://localhost:8080
PORT=8080
DATABASE_PATH=deadcomments.db
```

For the bundled Docusaurus test stand, seed a demo site:

```bash
DEADCOMMENTS_DEV_SEED=1 go run ./cmd/server
```

That creates a `docs-demo` site with automatic moderation and allowed origins for `localhost:3000` and `localhost:3001`.

## GitHub OAuth

Create a GitHub OAuth app:

- Homepage URL: `http://localhost:8080`
- Authorization callback URL: `http://localhost:8080/auth/github/callback`

Set:

```bash
GITHUB_CLIENT_ID=...
GITHUB_CLIENT_SECRET=...
GITHUB_ALLOWED_LOGINS=your-github-login
SERVER_SECRET="$(openssl rand -hex 32)"
SESSION_SECRET="$(openssl rand -hex 32)"
TRIPCODE_SECRET="$(openssl rand -hex 32)"
```

Only logins listed in `GITHUB_ALLOWED_LOGINS` can access `/admin`. If GitHub OAuth client credentials are configured, `GITHUB_ALLOWED_LOGINS` must contain at least one login; deadcomments fails closed instead of allowing arbitrary GitHub users.

For Docker Compose, copy `.env.example` to `.env` and fill in the same values. GitHub OAuth uses the client ID and client secret; a GitHub App ID is not required by deadcomments.

## Environment

| Variable | Default | Purpose |
| --- | --- | --- |
| `BASE_URL` | `http://localhost:8080` | Public base URL used for OAuth callback construction |
| `DATABASE_PATH` | `deadcomments.db` | SQLite database path |
| `DEADCOMMENTS_ENV` | empty | Set to `production` to require explicit stable secrets |
| `SERVER_SECRET` | generated on boot in development if empty | HMAC salt for IP, email, user-agent hashes, and embed submit tokens |
| `SESSION_SECRET` | `SERVER_SECRET` in development | Admin session token hashing and CSRF signing |
| `TRIPCODE_SECRET` | `SERVER_SECRET` in development | HMAC secret for public anonymous tripcodes |
| `GITHUB_CLIENT_ID` | empty | GitHub OAuth client ID |
| `GITHUB_CLIENT_SECRET` | empty | GitHub OAuth client secret |
| `GITHUB_ALLOWED_LOGINS` | empty | Comma-separated GitHub logins allowed to administer; required when OAuth is configured |
| `PORT` | `8080` | HTTP port |
| `SESSION_TTL_HOURS` | `720` | Admin session lifetime |
| `BEHIND_TRUSTED_PROXY` | `false` | Trust `X-Forwarded-For`/`X-Real-IP`; enable only behind a proxy that strips untrusted forwarded headers |
| `DEADCOMMENTS_DEV_SEED` | empty | Set to `1` to create the local demo site |

## Embed

```html
<div id="comments"></div>
<script
  src="https://comments.example.com/widget.js"
  data-site="my-blog"
  data-page="/posts/my-article"
  data-target="#comments"
  data-theme="auto">
</script>
```

Use explicit `data-page` keys. `data-page="auto"` is supported as a convenience and resolves to `location.pathname + location.search`.

Use `data-theme="inherit"` when embedding into a page whose colors should drive the widget. The host script reads the target container's computed text/background colors and sends safe theme variables into the iframe.

Public comment lists support `sort=oldest`, `sort=newest`, and `sort=best`. The `best` sort currently means the most active approved thread first; future reactions or ratings can replace that ranking signal without changing the public API.

## Docusaurus Test Stand

In one terminal:

```bash
DEADCOMMENTS_DEV_SEED=1 go run ./cmd/server
```

In another:

```bash
cd examples/docusaurus
npm install
npm start
```

Open `http://localhost:3000/docs/intro`, `threading`, and `moderation`. The example loader injects `http://localhost:8080/widget.js` using site key `docs-demo`.

### Docker Compose

To run the comments service and the Docusaurus test stand together:

```bash
docker compose up --build
```

Open:

- comments service: `http://localhost:8080`
- Docusaurus test stand: `http://localhost:3000/docs/intro`

The compose setup enables `DEADCOMMENTS_DEV_SEED=1`, so the `docs-demo` site is created automatically with allowed origins for local Docusaurus testing.

SQLite data is stored in the named Docker volume `deadcomments-data`. To reset local test data:

```bash
docker compose down -v
```

For admin GitHub OAuth inside Docker, export these before starting compose:

```bash
export GITHUB_CLIENT_ID=...
export GITHUB_CLIENT_SECRET=...
export GITHUB_ALLOWED_LOGINS=your-github-login
docker compose up --build
```

Or create a local `.env` from `.env.example`; Docker Compose will load it automatically. Never commit real OAuth secrets.

## Admin

Open `http://localhost:8080/admin`. From the admin panel you can:

- create and edit sites
- configure allowed origins and moderation mode
- inspect pages and change page state
- approve, reject, spam, delete, and edit comments
- manage reserved tripcode identities
- ban IP hashes and add word-ban rules

## Tripcode Identities

Tripcodes provide stable public identity hints without public user accounts. This is identity, not authentication.

Public commenters can write their name as:

```text
Display Name##secret
```

The server parses this into:

- display name: `Display Name`
- tripcode secret: `secret`

The submitted secret is never stored on the comment and is never returned to the frontend.

### Anonymous Tripcodes

If no reserved identity exists for the normalized display name, a submitted secret generates a stable public tripcode:

```text
base32(HMAC_SHA256(TRIPCODE_SECRET, normalized_secret))[0:10]
```

The public UI shows:

```text
Display Name ◆K7F3Q9M2PA
```

This means only that the same anonymous actor used the same secret. It does not prove the real-world identity of `Display Name`.

### Reserved Identities

Admins can create reserved identities at `/admin/identities`.

Reserved identity fields include:

- global or site-specific scope
- display name
- secret hash
- public tripcode
- badge type: `verified`, `admin`, `author`, `custom`
- optional badge label

If a comment uses a reserved name:

- no secret: rejected
- wrong secret: rejected
- correct secret: comment is attached to the reserved identity and rendered with the configured badge

This blocks simple spoofing of reserved names. For example, if `UT3USW` is reserved, `UT3USW` and `UT3USW##wrong` are rejected, while `UT3USW##correct-secret` is accepted.

If no site exists, the dashboard prompts you to create the first site.

## Storage and Security

SQLite migrations live in `migrations/`. The app enables foreign keys, WAL mode, and a busy timeout.

The service never stores raw IP addresses, raw user-agent strings, or raw commenter email addresses. IP, email, and user-agent values are HMAC-SHA256 hashes using `SERVER_SECRET`. If a commenter provides an email, deadcomments also stores a Gravatar-compatible MD5 avatar hash so the public widget can show an avatar without storing the email itself. Markdown is rendered with goldmark GFM and sanitized with bluemonday. Admin POST routes use CSRF tokens. Admin sessions use HttpOnly SameSite cookies.

Public comments can be arbitrarily deep in storage. The iframe renders root comments normally, visually flattens deeper replies under the root, and adds a `replying to @author` label instead of indenting forever.

## Events

Important domain actions publish durable events to the `events` table and record handler delivery state in `event_deliveries`. The current bus runs synchronous in-process handlers; future email, Discord, Telegram, webhook, or async worker handlers can subscribe without changing HTTP handlers.

Current event sources include site create/update, page auto-create and state changes, comment create/edit/status changes, IP bans, word bans, and admin login. The admin event log is available at `/admin/events`.

Identity create/update/delete and identity secret reset also publish events.

Comment moderation history is now an event subscriber: the audit handler listens to comment events and writes `moderation_events`. That keeps audit behavior out of handlers and away from core comment mutation code.

## Observability

Kubernetes-friendly status endpoints:

- `GET /livez`: process liveness, no database dependency
- `GET /readyz`: readiness check with SQLite ping, returns `503` if the database is unavailable
- `GET /healthz`: readiness-compatible alias for simple load balancers
- `GET /status`: JSON status payload with checks and server time

Prometheus metrics are exposed at:

```text
GET /metrics
```

The backend exports Go runtime/process metrics plus application metrics:

- `deadcomments_http_requests_total`
- `deadcomments_http_request_duration_seconds`
- `deadcomments_http_response_size_bytes`
- `deadcomments_http_requests_in_flight`
- `deadcomments_readiness_checks_total`
- `deadcomments_domain_events_total`
- `deadcomments_comments_created_total`
- `deadcomments_comment_moderation_actions_total`
- `deadcomments_page_events_total`
- `deadcomments_site_events_total`
- `deadcomments_ban_events_total`
- `deadcomments_identity_events_total`
- `deadcomments_admin_logins_total`

Business metrics are recorded by an event-bus subscriber, not directly from HTTP handlers or core mutation logic. Keep metric labels low-cardinality: status, action, route pattern, result, and bounded enum-like values are OK; site keys, page keys, comment IDs, IP hashes, user names, and free-form text are not OK as labels.

Example Kubernetes probes:

```yaml
livenessProbe:
  httpGet:
    path: /livez
    port: http
readinessProbe:
  httpGet:
    path: /readyz
    port: http
```

## Production Notes

- Run behind HTTPS and set `BASE_URL` to the HTTPS origin.
- Set stable, high-entropy `SERVER_SECRET`, `SESSION_SECRET`, and `TRIPCODE_SECRET` before accepting comments. HTTPS `BASE_URL` or `DEADCOMMENTS_ENV=production` requires these secrets explicitly.
- Keep `GITHUB_ALLOWED_LOGINS` explicit.
- Configure each site with exact allowed origins.
- Put the SQLite database on durable storage.
- Add process supervision with systemd, Docker, Nomad, Fly, or another single-service runtime.
- Terminate TLS at a reverse proxy and forward standard headers. Set `BEHIND_TRUSTED_PROXY=true` only when that proxy strips incoming `X-Forwarded-For` and `X-Real-IP` from untrusted clients.
- Expose `/metrics` only to Prometheus or a trusted internal network.
- Monitor SQLite file size, pending moderation volume, readiness failures, comment moderation outcomes, and HTTP 4xx/5xx rates.

## Backups

SQLite backups can be taken online:

```bash
sqlite3 deadcomments.db ".backup 'deadcomments-$(date +%Y%m%d-%H%M%S).db'"
```

Back up both the database and the exact `SERVER_SECRET`; without the secret, future hashes cannot be matched against existing bans or anonymous identities.

## Tests

```bash
go test ./...
```

The suite covers Markdown sanitization, origin validation, page auto-creation, comment and reply creation, moderation decisions, tree flattening, and banned-IP rejection.

## CI and Releases

GitHub Actions workflows live in `.github/workflows/`.

- `CI`: Go formatting, module consistency, vet, race tests, coverage artifact, `govulncheck`, Docusaurus build, and Docker Compose smoke test.
- `CodeQL`: scheduled and PR security analysis for Go and JavaScript.
- `Release`: triggered by tags matching `v*.*.*`.

To publish a release:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The release workflow:

- runs the test gate
- builds Linux, macOS, and Windows release archives
- includes the binary plus migrations/templates/static/widget runtime assets
- creates SHA256 checksums
- publishes a GitHub Release
- publishes a multi-arch Docker image to GHCR

Dependabot is configured for GitHub Actions, Go modules, Docker base images, and the Docusaurus test stand.
