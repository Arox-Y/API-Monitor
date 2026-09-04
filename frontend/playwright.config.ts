import { defineConfig, devices } from "@playwright/test";

/**
 * Playwright configuration for the end-to-end suite.
 *
 * These tests drive a REAL browser against the FULL stack: frontend, backend
 * and database, all actually running. That is what makes them able to catch
 * bugs no unit test can - a wrong CORS header, a cookie that does not survive
 * a redirect, a button that is visually present but not clickable.
 *
 * The cost is speed and fragility, which is why there is exactly ONE journey
 * here rather than many. A large E2E suite becomes slow and flaky enough that
 * people stop trusting red, which defeats the purpose of testing at all.
 */
export default defineConfig({
  testDir: "./e2e",

  // Generous. The journey waits on the real background worker TWICE - once
  // for the initial check, once for recovery - and each wait can legitimately
  // take up to (tick interval + monitor interval), around 40s worst case.
  // A first attempt at 90s hit the outer timeout while the individual waits
  // were still within their own budgets - the steps really do take this long
  // together, it is not a hang.
  timeout: 180_000,

  // Never allow accidental parallelism for this suite: every test shares one
  // set of running services and, in the worst case, one browser session. Two
  // tests running at once could interfere with each other's state.
  fullyParallel: false,
  workers: 1,

  // Retries mask real bugs by making a failure look like a fluke. None here -
  // if it is flaky, that IS the bug to fix.
  retries: 0,

  reporter: [["list"]],

  use: {
    baseURL: "http://localhost:3000",
    // On the first failure of each test, keep the trace so a screenshot and
    // full timeline can be inspected afterwards instead of trying to
    // reproduce a flake blind.
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },

  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],

  // We do NOT use Playwright's webServer option to start the app.
  //
  // The critical journey needs three things running together: Postgres, the
  // Go backend, and the Next.js frontend. Playwright's built-in server
  // management is built for one process, not a stack - so all three are
  // started by hand before running these tests. See e2e/README.md.
});
