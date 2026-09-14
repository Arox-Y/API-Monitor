# syntax=docker/dockerfile:1

# ---------------------------------------------------------------------------
# Stage 1: deps
#
# Installing dependencies is kept in its own stage, separate from copying the
# rest of the source. Docker caches layers, and this one only invalidates
# when package.json/package-lock.json change - editing a component does not
# force a full npm install to run again on the next build.
# ---------------------------------------------------------------------------
FROM node:20-alpine AS deps

WORKDIR /app

COPY package.json package-lock.json ./
# `npm ci` rather than `npm install`: it installs EXACTLY what package-lock.json
# says, and refuses if the lockfile and package.json disagree. `npm install`
# would happily "fix" a mismatch by changing the lockfile, which is precisely
# the kind of surprise a reproducible build cannot afford.
RUN npm ci

# ---------------------------------------------------------------------------
# Stage 2: builder
# ---------------------------------------------------------------------------
FROM node:20-alpine AS builder

WORKDIR /app

COPY --from=deps /app/node_modules ./node_modules
COPY . .

# NEXT_PUBLIC_* variables are baked into the CLIENT-SIDE JavaScript bundle at
# `next build` time - not read at runtime like a normal server-side env var.
# That means it has to arrive as a build ARG here, not as a plain `environment:`
# entry on the runtime container in compose (see docker-compose.yml, which
# passes this through as `build.args`).
ARG NEXT_PUBLIC_API_URL=http://localhost:8080
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL

RUN npm run build

# ---------------------------------------------------------------------------
# Stage 3: runner
#
# next.config.ts sets `output: "standalone"`, which makes `next build`
# produce .next/standalone: a self-contained server with only the production
# dependencies that survived tree-shaking already copied in. Without that
# setting, this stage would need the entire node_modules tree - hundreds of
# MB including every devDependency - just to run `next start`.
# ---------------------------------------------------------------------------
FROM node:20-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production

RUN addgroup -S nextjs && adduser -S nextjs -G nextjs

# Static assets (images, the favicon) are served directly and are not part
# of the standalone bundle, so they are copied across explicitly.
COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nextjs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nextjs /app/.next/static ./.next/static

USER nextjs

EXPOSE 3000
ENV PORT=3000

# The standalone output includes its own minimal server.js - no `next start`,
# no full Next.js CLI needed at runtime.
CMD ["node", "server.js"]
