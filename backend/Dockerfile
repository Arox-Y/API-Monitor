# syntax=docker/dockerfile:1

# ---------------------------------------------------------------------------
# Stage 1: build
#
# A MULTI-STAGE build. This whole stage - the Go toolchain, the full module
# cache, every *_test.go file - exists only to produce one file: a compiled
# binary. None of it ends up in the image we actually ship; stage 2 starts
# from a clean base and copies across only that one file.
#
# The Go version here should track go.mod's `go 1.26.3` line.
# ---------------------------------------------------------------------------
FROM golang:1.26-alpine AS build

WORKDIR /src

# Copying go.mod/go.sum BEFORE the rest of the source, and running
# `go mod download` before `COPY . .`, is deliberate layer ordering.
#
# Docker caches each instruction as a layer and reuses a cached layer if
# nothing that produced it changed. Dependencies change far less often than
# application code, so keeping this step separate means editing a .go file
# does not force every dependency to be re-downloaded on the next build.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 produces a fully static binary with no dependency on the
# system's C library. Combined with a minimal runtime base below, that is
# what lets the final image contain almost nothing but our own code.
#
# -ldflags="-s -w" strips debug symbols the running binary never needs,
# which noticeably shrinks it.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/api ./cmd/api

# ---------------------------------------------------------------------------
# Stage 2: runtime
#
# Alpine rather than `scratch` (an entirely empty image) for one concrete
# reason: our own code makes outbound HTTPS requests to whatever URLs users
# register as monitors, and verifying a TLS certificate requires a set of
# trusted root certificates on disk somewhere. `scratch` has no filesystem at
# all, so TLS to any monitored HTTPS endpoint would fail immediately with
# "x509: certificate signed by unknown authority". Alpine is still tiny
# (a few MB) and lets us add exactly the one package that fixes this.
# ---------------------------------------------------------------------------
FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates

# Running as root inside a container is the same bad idea it is on a real
# machine: if the process is ever compromised, root inside the container is a
# meaningfully larger blast radius than an unprivileged user. Creating a
# dedicated, unprivileged user costs three lines.
RUN addgroup -S apimonitor && adduser -S apimonitor -G apimonitor

COPY --from=build /bin/api /usr/local/bin/api

USER apimonitor

EXPOSE 8080

# wget is part of busybox, already present in every Alpine image - no
# extra package needed for a healthcheck. --spider means "check it responds,
# discard the body".
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=5 \
    CMD wget --spider -q http://localhost:8080/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/api"]
