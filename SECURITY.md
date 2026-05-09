# Security Policy

## Supported Versions

`deadcomments` is early-stage software. Security fixes are applied to the `main` branch and to the latest tagged release once releases begin.

## Reporting a Vulnerability

Please do not open a public issue for sensitive security reports.

Report vulnerabilities privately through GitHub Security Advisories for this repository when available, or contact the repository owner directly through GitHub.

Useful report details:

- affected version or commit
- deployment mode
- reproduction steps
- expected impact
- logs or screenshots, if safe to share

## Security Expectations

Security-sensitive areas include:

- Markdown rendering and sanitization
- admin GitHub OAuth
- admin sessions and CSRF
- public origin validation
- IP, email, user-agent, and tripcode hashing
- event payload contents
- Docker/runtime configuration

Never include real secrets, OAuth tokens, session cookies, raw IP addresses, raw emails, raw user agents, or tripcode secrets in bug reports.

