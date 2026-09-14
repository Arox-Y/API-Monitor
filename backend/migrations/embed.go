// Package migrations embeds every SQL migration file into the compiled
// binary.
//
// WHY EMBED THEM.
//
// Without this, running migrations in production means either mounting the
// source files into the container as a volume, or copying them alongside the
// binary and hoping the path is right at runtime. Go's embed package lets the
// *.sql files become part of the binary itself at build time - the container
// image can contain nothing but that one binary and still carry its own
// schema with it. This is what makes RunMigrations (internal/database) work
// identically on a developer's machine and inside a minimal Docker image.
package migrations

import "embed"

// FS holds every file in this directory matching *.sql, embedded by the
// go:embed directive below. embed.FS is a read-only filesystem baked into
// the binary at compile time - there is no runtime file I/O involved at all.
//
//go:embed *.sql
var FS embed.FS
