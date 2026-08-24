-- Migration 000001 (DOWN) — undo everything 000001_init_schema.up.sql created.
--
-- Every "up" migration needs a matching "down" so a change can be reversed.
--
-- Order is the REVERSE of the up file. A table cannot be dropped while another
-- table still points at it, so children are dropped before their parents:
--   checks / incidents -> monitors -> projects -> users
--
-- Dropping a table also drops its indexes automatically, so they need no
-- separate DROP statements here.
--
-- IF EXISTS means "do not error if it is already gone", which makes this file
-- safe to run even if a previous attempt failed halfway through.

DROP TABLE IF EXISTS incidents;
DROP TABLE IF EXISTS checks;
DROP TABLE IF EXISTS monitors;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS users;
