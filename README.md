# API Monitor

Uptime and health monitoring for APIs and websites. Register the endpoints you care
about, and the system checks them on a schedule, recording availability, response times,
failures, and incidents — with an optional public status page per project.

> **🚧 In development.** The database, backend service, and dashboard UI are running.
> Authentication, the monitoring worker, and incident detection are next.
> See [Current status](#current-status) for exactly what works today.

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

Docker & Docker Compose · Go 1.26+ · Node.js 20+

### Run it

```bash
git clone https://github.com/Arox-Y/API-Monitor.git
cd API-Monitor
cp .env.example .env      # set JWT_SECRET — there is deliberately no default
```

**1. Database**

```bash
docker compose up -d db
```

Postgres is published on host port **5433**, not 5432, to avoid clashing with a locally
installed PostgreSQL. Inside the compose network the backend reaches it as `db:5432`.

**2. Apply the schema**

```bash
docker compose exec -T db psql -U apimonitor -d apimonitor \
  < backend/migrations/000001_init_schema.up.sql
```

**3. Backend**

```bash
cd backend && go run ./cmd/api
```

Health check: <http://localhost:8080/healthz>

**4. Frontend**

```bash
cd frontend && npm install && npm run dev
```

Open <http://localhost:3000>

---

## Project layout

```
├── backend/
│   ├── cmd/api/            entry point: config, database, server, shutdown
│   ├── internal/
│   │   ├── config/         environment → validated struct
│   │   └── database/       connection pool, ping, close
│   └── migrations/         versioned schema, up and down
└── frontend/
    └── src/
        ├── app/            routes: landing, (auth), (dashboard)
        ├── components/     ui/ (shadcn) · app/ (product) · landing/ (marketing)
        ├── lib/            formatting helpers
        └── types/          API response types
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
PostgreSQL enforces it rather than a service method.

**Incidents are created by transitions, not states.** Only the healthy→failing edge opens
one; repeated failures do not.

**Versioned SQL migrations, not `AutoMigrate`.** GORM's `AutoMigrate` cannot express the
partial unique index above, nor `CHECK` constraints, and offers no down migration or
reviewable diff. Schema lives in numbered `.up.sql`/`.down.sql` pairs.

**`checks.status_code` is nullable.** A timeout or DNS failure produces no status code at
all. `NULL` says that honestly; `0` would be a lie and `200` would be dangerous.

**Validate in the application *and* constrain in the database.** They protect against
different failures — one guards against bad user input, the other against bad code.

**404, not 403, for another user's resource.** A `403` confirms the resource exists,
letting an attacker enumerate IDs. Ownership is scoped at the service layer, once.

**JWT in an httpOnly cookie, not `localStorage`.** JavaScript cannot read it, so an XSS
cannot steal the session. The trade-off is that CSRF becomes the relevant threat,
mitigated with `SameSite=Lax`.

**A health check that really checks.** `/healthz` pings the database and returns 503 when
it is unreachable. A health check that always returns `ok` reports healthy while the
application is broken.

---

## Current status

| Area | State |
|---|---|
| Database schema (5 tables, 13 indexes) | ✅ Applied and constraint-tested |
| Docker Compose (PostgreSQL) | ✅ Running with healthcheck |
| Go backend skeleton + `/healthz` | ✅ Verified against a live database |
| Marketing landing page | ✅ Complete |
| Dashboard UI | ✅ Built — currently on sample data |
| Authentication | ⬜ Next |
| Projects & monitors CRUD | ⬜ Next |
| Monitoring worker | ⬜ Planned |
| Incidents & status page | ⬜ Planned |
| Tests | ⬜ Planned |

The dashboard renders sample data and says so on screen. Nothing is presented as working
that isn't.

---

## Known limitations

Stated deliberately rather than discovered later:

- One failed check opens an incident — no flap threshold yet
- Correct with exactly **one** backend instance; the scheduler has no leader election.
  The fix (`SELECT … FOR UPDATE SKIP LOCKED`) is understood but not implemented
- No alerting — incidents are recorded and visible, but nothing notifies you
- No retention job; the `checks` table grows unbounded
- SSRF mitigation is not yet implemented — the service fetches user-supplied URLs

---

## License

Not yet specified.
