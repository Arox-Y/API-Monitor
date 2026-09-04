# End-to-end tests

One test: the critical user journey from `docs/PRODUCT_SPEC.md` §5, driven
through a real Chromium browser against the full running stack.

## Why only one test

E2E tests are slow and, compared to a unit test, more prone to flaking. A
large suite of them tends to erode trust in red until failures get ignored -
which defeats the point of testing at all. One test covering the path that
must never break is worth more than many covering edge cases a unit test
would catch just as well, faster.

## Running it

Playwright's `webServer` option starts *one* process for you. This project
needs three running together - Postgres, the Go backend, the Next.js frontend
- so they are started by hand instead:

```bash
# Terminal 1
docker compose up -d db

# Terminal 2
cd backend && go run ./cmd/api

# Terminal 3
cd frontend && npm run dev

# Terminal 4 - once all three are up
cd frontend && npm run test:e2e
```

## How determinism is achieved without test-only code

Two problems, two decisions:

**The outage.** The monitor points at a tiny local HTTP server the test
starts itself (`node:http`, listening on a random free port) and can flip
between returning 200 and 503 on command. A real external URL would make the
test's result depend on whether that site happens to be up right now - the
outage the test *wants* to create becomes indistinguishable from an unrelated
network problem.

**Waiting for the worker.** The scheduler checks on its own schedule; nothing
in the test forces a check early. `TESTING.md` considered and rejected a
test-only endpoint to trigger one, because that means test-only code sitting
in a production code path - a real risk if it is ever left reachable.
Instead the test polls with Playwright's own retrying assertions
(`expect(...).toBeVisible({ timeout: 60_000 })`), which is the idiomatic
approach and adds nothing to the app itself.

If this proves flaky in practice, the tradeoff gets reassessed - a flaky E2E
test is worse than no E2E test, because it trains you to ignore failures.

## Status

Written and reviewed against the actual component markup (label text, link
hrefs, button names) but **not yet executed** - it depends on Docker, which
was not running in this environment when this test was authored. Run it
before trusting it.
