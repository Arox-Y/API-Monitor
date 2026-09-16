# API Monitor

Uptime and health monitoring for APIs and websites. Register the endpoints you care
about, and the system checks them on a schedule, recording availability, response times,
failures, and incidents — with an optional public status page per project.

> **Status: MVP complete.** Register, sign in, create projects and monitors, watch the
> background worker check them for real, see incidents open and close automatically, and
> publish a public status page — all working end to end, backed by 128 Go tests, 25
> frontend tests, and a Playwright end-to-end test that passes against the real Docker
> stack. See [Current status](#current-status) for exactly what is verified and what is
> not.

---

## What it does

- **Monitors** — watch any HTTP endpoint on its own interval, timeout, and expected status code
- **Checks** — every result recorded: success, HTTP status, response time, error
- **Incidents** — opened automatically when a monitor starts failing, closed on recovery, with duration
- **Dashboard** — monitors up/down, 24-hour uptime, recent incidents and checks
- **Status page** — a public, unauthenticated page per project

---

## Tech stack

**Frontend** — Next.js 15 (App Router) · TypeScript · Tailwind CSS v4 · shadcn/ui ·
TanStack Query · React Hook Form · Zod · Recharts

**Backend** — Go 1.26 · Gin · PostgreSQL 16 · GORM · JWT

**Infrastructure** — Docker · Docker Compose

---

## Architecture

A **modular monolith**: one Go binary serves the HTTP API *and* runs the background
monitoring worker, against one PostgreSQL database.

```
Next.js ──HTTP/JSON──▶ Go backend ──▶ PostgreSQL
                        ├─ API (Gin)
                        └─ Worker  ──▶ monitored URLs
```

The worker ticks on a fixed interval, queries which monitors are due, and checks them
through a concurrency-bounded goroutine pool. Each result — the check row, the monitor's
status, and any incident opened or closed — is written inside **one transaction**.

Two services would have meant a message queue, two deployments, and distributed
debugging, to solve an independent-scaling problem this project does not have. The worker
is a package, not a service; the boundary is clean enough to extract later if needed.

---

## Getting started

### Prerequisites

Docker & Docker Compose · Go 1.26+ (for local backend development) · Node.js 20+ (for
local frontend development)

### Run it — one command

```bash
git clone https://github.com/Arox-Y/API-Monitor.git
cd API-Monitor
cp .env.example .env      # set JWT_SECRET — there is deliberately no default
docker compose up
```

This builds and starts the database, the backend, and the frontend together. The backend
applies its own schema on first start — there is no separate migration step to run by
hand.

| Service | URL |
|---|---|
| Frontend | <http://localhost:3000> |
| Backend API | <http://localhost:8080> |
| Health check | <http://localhost:8080/healthz> |

> The database's host port defaults to **5433**, not 5432, to avoid clashing with a
> locally installed PostgreSQL. Inside the Compose network the backend reaches it as
> `db:5432` regardless.

### Local development, without Docker for the app itself

Useful for editing Go or TypeScript with fast rebuilds, while still using Docker for the
database only:

```bash
docker compose up -d db                # database only
cd backend && go run ./cmd/api          # backend — applies its own schema
cd frontend && npm install && npm run dev
```

---

## Project layout

```
├── backend/
│   ├── cmd/api/            entry point: config, database, migrations, server, worker
│   ├── internal/
│   │   ├── auth/           password hashing, JWT
│   │   ├── config/         environment → validated struct
│   │   ├── database/       connection pool, migration runner
│   │   ├── handler/        HTTP layer
│   │   ├── middleware/     auth, CORS, rate limiting, security headers
│   │   ├── models/         data shapes
│   │   ├── repository/     database queries, always ownership-scoped
│   │   ├── router/         the entire route table, in one file
│   │   ├── service/        business rules, validation, authorization
│   │   ├── testutil/       integration-test database helpers
│   │   └── worker/         the scheduler, the HTTP checker, the SSRF guard
│   ├── migrations/         versioned schema, embedded into the binary
│   └── Dockerfile          multi-stage: Go build → minimal Alpine runtime
└── frontend/
    └── src/
        ├── app/            routes: landing, (auth), (dashboard), status/[slug]
        ├── components/     ui/ (shadcn) · app/ (product) · landing/ (marketing)
        ├── hooks/          TanStack Query hooks, one per resource
        ├── lib/            API client, Zod schemas, formatting helpers
        └── types/          API response types
    ├── e2e/                Playwright: the critical user journey, end to end
    └── Dockerfile          multi-stage: Next.js standalone build → minimal runtime
```

---

## Engineering notes

A few decisions worth calling out, with their trade-offs rather than just the choice.

**At most one open incident per monitor, enforced by the database.**

```sql
CREATE UNIQUE INDEX idx_incidents_one_open
  ON incidents (monitor_id) WHERE resolved_at IS NULL;
```

A partial unique index: unique only across *unresolved* rows. A monitor can accumulate
unlimited closed incidents but never two open ones. An hour of downtime becomes one
incident, not sixty — and the guarantee survives bugs in application code, because
PostgreSQL enforces it rather than a service method. Proven by a test that forces a
transaction to fail partway through and asserts nothing was left behind.

**Incidents are created by transitions, not states.** Only the healthy→failing edge opens
one; repeated failures do not.

**SSRF protection lives at the network dial, not at URL validation.** The product's job is
"fetch a URL a user gave us" — a user could point a monitor at `http://169.254.169.254/`
(cloud metadata) or `http://localhost:5432`. A one-time check when the monitor is saved
cannot catch DNS rebinding — the hostname resolving somewhere different by the time it is
actually checked. A `net.Dialer.Control` callback inspects the *resolved IP*, at the
moment of every real connection attempt, which closes that gap and covers every HTTP
redirect through the same mechanism, for free. It only runs in production (`ENV=production`);
in development it's off, the same way the login cookie's `Secure` flag is relaxed, so a
monitor can legitimately target something on `localhost` while you're building.

**Versioned SQL migrations, not `AutoMigrate`.** GORM's `AutoMigrate` cannot express the
partial unique index above, nor `CHECK` constraints, and offers no down migration or
reviewable diff. Schema lives in numbered `.up.sql`/`.down.sql` pairs, embedded into the
compiled binary with Go's `embed` package and applied automatically at startup — a fresh
`docker compose up` needs no manual step.

**`checks.status_code` is nullable.** A timeout or DNS failure produces no status code at
all. `NULL` says that honestly; `0` would be a lie and `200` would be dangerous.

**Validate in the application *and* constrain in the database.** They protect against
different failures — one guards against bad user input, the other against bad code.

**404, not 403, for another user's resource.** A `403` confirms the resource exists,
letting an attacker enumerate IDs. Ownership is scoped at the repository layer on every
query, proven by a test suite that deliberately breaks that scoping and confirms the test
catches it before trusting the test at all.

**JWT in an httpOnly cookie, not `localStorage`.** JavaScript cannot read it, so an XSS
cannot steal the session. The trade-off is that CSRF becomes the relevant threat,
mitigated with `SameSite=Lax`.

**A health check that really checks.** `/healthz` pings the database and returns 503 when
it is unreachable. A health check that always returns `ok` reports healthy while the
application is broken.

**Structured logs never carry a monitor's URL.** A monitor's URL is user-supplied and can
carry a secret in its query string (an API key used as a health-check credential).
Incident logs identify the monitor by name and ID instead.

---

## Testing

```bash
# Backend
cd backend
go test ./...                    # unit tests — fast, no database
go test -tags=integration ./...  # + real-database tests (needs `docker compose up -d db`)

# Frontend
cd frontend
npm test                         # vitest
npx playwright test              # end-to-end, needs the full stack running
```

## Current status

| Area | State |
|---|---|
| Authentication (register, login, logout, sessions) | ✅ 38 tests |
| Projects & monitors CRUD, ownership-scoped | ✅ cross-user isolation proven by test |
| Monitoring worker | ✅ verified against a real outage end to end |
| Incidents (open/close, one per outage) | ✅ |
| Public status page | ✅ leak-audited — no URLs, IDs, or internal errors exposed |
| Dashboard | ✅ real aggregates, computed in SQL |
| SSRF protection, rate limiting, security headers | ✅ |
| Structured logging | ✅ |
| Dark mode | ✅ |
| Backend tests | ✅ 128 passing, including `go test -race` (run in a Linux container — this machine's own gcc is 32-bit) |
| Frontend tests | ✅ 25 passing |
| End-to-end test | ✅ all 10 steps pass, against the real Docker stack |
| Docker images (backend + frontend) | ✅ build-verified, full stack runs via `docker compose up` |
| Responsive layout (375/768/1440px) | ✅ reviewed via captured screenshots, no defects |
| Accessibility (WCAG 2.0/2.1 A+AA, automated scan) | ✅ 0 violations across all pages |

---

## Known limitations

Stated deliberately rather than discovered later:

- One failed check opens an incident — no flap threshold yet
- Correct with exactly **one** backend instance; the scheduler has no leader election.
  The fix (`SELECT … FOR UPDATE SKIP LOCKED`) is understood but not implemented
- No alerting — incidents are recorded and visible, but nothing notifies you
- No retention job; the `checks` table grows unbounded
- Rate limiting is per-process, in memory — resets on restart, not shared across instances
  (consistent with the single-instance limitation above)

---

## License

Not yet specified.
