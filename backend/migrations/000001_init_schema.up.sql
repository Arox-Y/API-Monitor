-- Migration 000001 (UP) — create the initial schema.
--
-- Creates the 5 tables described in docs/DATABASE_DESIGN.md.
-- Order matters: a table must exist before another table can reference it.
--   users -> projects -> monitors -> checks / incidents

-- ---------------------------------------------------------------------------
-- users — one row per account
-- ---------------------------------------------------------------------------
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- UNIQUE stops two accounts sharing an email address.
    -- We always lowercase the email before saving, because UNIQUE in Postgres
    -- is case-SENSITIVE: without lowercasing, Bob@x.com and bob@x.com would
    -- both be allowed as separate accounts.
    email         TEXT NOT NULL UNIQUE,

    -- Named password_hash, never "password", because we never store the real
    -- password. We store a scrambled version that cannot be reversed.
    password_hash TEXT NOT NULL,

    name          TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------------------
-- projects — a group of monitors, e.g. "Acme API"
-- ---------------------------------------------------------------------------
CREATE TABLE projects (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- REFERENCES users(id) means: this must point at a real user.
    -- ON DELETE CASCADE means: if that user is deleted, delete their projects
    -- too. The database enforces this, so no orphaned rows can be left behind.
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',

    -- slug: the short web-safe name used in the public URL, /status/acme-api
    slug        TEXT NOT NULL UNIQUE,

    -- is_public: the on/off switch for the public status page
    is_public   BOOLEAN NOT NULL DEFAULT false,

    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- An index is like the index at the back of a book: it lets Postgres jump
-- straight to the rows it needs instead of reading every row.
-- We add one here because "show me this user's projects" runs constantly.
CREATE INDEX idx_projects_user_id ON projects (user_id);

-- ---------------------------------------------------------------------------
-- monitors — one URL being watched
-- ---------------------------------------------------------------------------
CREATE TABLE monitors (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id       UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,

    -- ---- the 7 settings the user controls ----
    name             TEXT NOT NULL,
    url              TEXT NOT NULL,

    -- CHECK is a rule the database refuses to break. Even if our Go code has a
    -- bug, or someone edits the row by hand, an invalid value is rejected.
    method           TEXT NOT NULL DEFAULT 'GET'
                     CHECK (method IN ('GET', 'POST', 'HEAD')),

    interval_seconds INTEGER NOT NULL DEFAULT 60
                     CHECK (interval_seconds >= 30),

    timeout_seconds  INTEGER NOT NULL DEFAULT 10
                     CHECK (timeout_seconds BETWEEN 1 AND 30),

    expected_status  INTEGER NOT NULL DEFAULT 200
                     CHECK (expected_status BETWEEN 100 AND 599),

    enabled          BOOLEAN NOT NULL DEFAULT true,

    -- ---- state the app manages, not the user ----

    -- status: a sticky note saying how this monitor is doing right now, so the
    -- dashboard can count up/down instantly instead of scanning every check.
    -- 'pending' = created but never checked yet.
    status           TEXT NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending', 'up', 'down')),

    -- last_checked_at: when we last visited this URL. NULL means never.
    -- The background worker uses this to work out who is due for a check.
    last_checked_at  TIMESTAMPTZ,

    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_monitors_project_id ON monitors (project_id);

-- Used by the background worker's "which monitors are due?" question.
CREATE INDEX idx_monitors_due ON monitors (enabled, last_checked_at);

-- ---------------------------------------------------------------------------
-- checks — the result of ONE visit to a URL. The biggest table by far:
-- one monitor checked every 60s produces ~1,440 rows per day.
-- ---------------------------------------------------------------------------
CREATE TABLE checks (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id       UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,

    checked_at       TIMESTAMPTZ NOT NULL,
    success          BOOLEAN NOT NULL,

    -- Nullable on purpose. If the site times out or the domain does not
    -- resolve, there IS no status code. NULL means "no reply at all".
    -- Storing 0 would be a lie; storing 200 would be dangerously wrong.
    status_code      INTEGER,

    response_time_ms INTEGER NOT NULL,
    error            TEXT NOT NULL DEFAULT ''

    -- Note: no updated_at. A check is a historical fact. It never changes.
);

-- Two columns in ONE index, and the order matters.
-- Every question is "recent checks FOR ONE MONITOR": filter by monitor_id,
-- then sort by newest first. Matching that order lets Postgres read the rows
-- already sorted, with no extra sorting step.
CREATE INDEX idx_checks_monitor_checked_at ON checks (monitor_id, checked_at DESC);

-- ---------------------------------------------------------------------------
-- incidents — a period of time during which a monitor was failing
-- ---------------------------------------------------------------------------
CREATE TABLE incidents (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id       UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,

    started_at       TIMESTAMPTZ NOT NULL,

    -- NULL means the incident is STILL OPEN (the site is still down).
    resolved_at      TIMESTAMPTZ,

    duration_seconds INTEGER,
    cause            TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_incidents_monitor_started ON incidents (monitor_id, started_at DESC);

-- The most important line in this file.
--
-- A "partial unique index": unique, but only for rows matching the WHERE.
-- It means a monitor can have unlimited CLOSED incidents in its history, but
-- never two OPEN ones at the same time.
--
-- Why it matters: this rule is now enforced by PostgreSQL itself. Even if our
-- Go code has a bug and tries to open a second incident, the database will
-- reject it. Rules enforced by the database survive bugs in code.
CREATE UNIQUE INDEX idx_incidents_one_open
    ON incidents (monitor_id)
    WHERE resolved_at IS NULL;
