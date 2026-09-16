import { expect, test, type BrowserContext, type Page } from "@playwright/test";
import http from "node:http";

/**
 * The critical user journey (docs/PRODUCT_SPEC.md §5), driven through a real
 * browser against the full running stack: Postgres, the Go backend, and the
 * Next.js frontend.
 *
 * This is the ONLY end-to-end test in the project, and that is deliberate.
 * E2E tests are slow, and a whole suite of them tends to become flaky enough
 * that people learn to ignore red - which defeats the point of testing at
 * all. One test, covering the path that must never break, is worth more than
 * twenty covering edge cases better suited to a unit test.
 */

// ---------------------------------------------------------------------------
// A tiny target server this test fully controls.
//
// Pointing a monitor at a real external site would make the test's result
// depend on whether that site happens to be reachable right now - the outage
// the test WANTS to create becomes indistinguishable from an unrelated
// network hiccup. This server lets the test flip between healthy and broken
// on command, so the outage is deterministic.
// ---------------------------------------------------------------------------
let targetHealthy = true;
let targetServer: http.Server;
let targetPort: number;

// The backend that checks this server is not always this same machine's
// network namespace: run the stack via `docker compose up` and the worker
// lives inside a container, where "127.0.0.1" means the container itself,
// not this host. E2E_TARGET_HOST lets the same test address either setup -
// "127.0.0.1" (default) for a backend run with `go run ./cmd/api`,
// "host.docker.internal" for a backend running in Docker.
const TARGET_HOST = process.env.E2E_TARGET_HOST ?? "127.0.0.1";

test.beforeAll(async () => {
  targetServer = http.createServer((_req, res) => {
    if (targetHealthy) {
      res.writeHead(200, { "Content-Type": "text/plain" });
      res.end("ok");
    } else {
      res.writeHead(503, { "Content-Type": "text/plain" });
      res.end("down for testing");
    }
  });

  // Bind to every interface unless we're in the plain-loopback case: a
  // container reaching in via host.docker.internal cannot connect to a
  // server bound only to the host's loopback interface.
  const bindHost = TARGET_HOST === "127.0.0.1" ? "127.0.0.1" : "0.0.0.0";
  await new Promise<void>((resolve) => {
    targetServer.listen(0, bindHost, resolve);
  });

  const address = targetServer.address();
  if (typeof address !== "object" || address === null) {
    throw new Error("could not determine the target server's port");
  }
  targetPort = address.port;
});

test.afterAll(async () => {
  await new Promise<void>((resolve, reject) => {
    targetServer.close((err) => (err ? reject(err) : resolve()));
  });
});

test("register through public status page", async ({ page, browser }) => {
  // A fresh identity every run, so re-running the suite never collides with a
  // previous run's leftover account.
  const unique = Date.now();
  const email = `e2e-${unique}@example.com`;
  const password = "a-strong-e2e-password";
  const projectName = `E2E Project ${unique}`;
  const monitorName = `E2E Monitor ${unique}`;
  const targetUrl = `http://${TARGET_HOST}:${targetPort}/health`;

  await test.step("1. register", async () => {
    await page.goto("/register");
    await page.getByLabel("Name").fill("E2E Test User");
    await page.getByLabel("Email").fill(email);
    await page.getByLabel("Password").fill(password);
    await page.getByRole("button", { name: "Create account" }).click();

    // Registration signs the user in immediately and redirects.
    await expect(page).toHaveURL("/dashboard", { timeout: 15_000 });
  });

  await test.step("2. login (session survives a reload)", async () => {
    // Registration already proved the cookie is set; reloading proves it is
    // actually being read back rather than the page relying on client state
    // that a real navigation would lose.
    await page.reload();
    await expect(page).toHaveURL("/dashboard");
  });

  await test.step("3. create a project", async () => {
    await page.goto("/projects");
    await page.getByRole("button", { name: "New project" }).click();
    await page.getByLabel("Name").fill(projectName);
    await page.getByRole("button", { name: "Create project" }).click();
    await expect(page.getByText(projectName)).toBeVisible();
  });

  await test.step("4. add a monitor", async () => {
    await page.goto("/monitors");
    await page.getByRole("button", { name: "New monitor" }).click();
    // The project select already defaults to the only project that exists.
    await page.getByLabel("Name").fill(monitorName);
    await page.getByLabel("URL").fill(targetUrl);
    await page.getByLabel("Interval (s)").fill("30");
    await page.getByLabel("Timeout (s)").fill("5");
    await page.getByRole("button", { name: "Create monitor" }).click();
    await expect(page.getByText(monitorName)).toBeVisible();
  });

  await test.step("5. automatic monitoring", async () => {
    // No test-only endpoint forces a check - that would be code that exists
    // solely to make a test pass, which TESTING.md rules out. Instead this
    // polls, via an assertion with a generous timeout, for the worker's own
    // schedule to reach the monitor.
    await page.goto("/dashboard");
    await page.getByRole("link", { name: monitorName }).click();
    await expect(page).toHaveURL(/\/monitors\//);

    await expect(page.getByText("Up", { exact: true })).toBeVisible({
      timeout: 60_000,
    });
  });

  await test.step("6. response times are recorded", async () => {
    await expect(page.getByText(/checks · last 24h/)).toBeVisible();
  });

  await test.step("7. check history is visible", async () => {
    await expect(
      page.getByRole("cell", { name: "Success" }).first(),
    ).toBeVisible();
  });

  await test.step("8. an incident is detected", async () => {
    targetHealthy = false;

    await expect(page.getByText("Down", { exact: true })).toBeVisible({
      timeout: 60_000,
    });

    await page.reload();
    await expect(page.getByText("Ongoing")).toBeVisible();
  });

  await test.step("9. incident history shows recovery", async () => {
    targetHealthy = true;

    await expect(page.getByText("Up", { exact: true })).toBeVisible({
      timeout: 60_000,
    });

    await page.reload();
    // Resolved incidents show their duration rather than "Ongoing".
    await expect(page.getByText(/Resolved ·/)).toBeVisible({
      timeout: 15_000,
    });
  });

  await test.step("10. the public status page, with NO session at all", async () => {
    await page.goto("/projects");

    const projectRow = page.locator("li", { hasText: projectName });
    await projectRow.getByRole("switch").click();

    const statusLink = projectRow.getByRole("link", { name: /\/status\// });
    const href = await statusLink.getAttribute("href");
    if (!href) throw new Error("the status page link had no href");

    // A SEPARATE browser context, not just a new tab. A new tab in the same
    // context would still carry the session cookie, silently proving nothing
    // about whether the page truly works for an anonymous visitor.
    const anonymousContext: BrowserContext = await browser.newContext();
    const anonymousPage: Page = await anonymousContext.newPage();

    await anonymousPage.goto(href);

    // getByRole("heading", ...), not getByText: the project name also appears
    // in the page's <title> (set by generateMetadata for the browser tab),
    // and getByText matched both - a strict-mode violation. Scoping to the
    // visible <h1> is both more precise and closer to what a visitor actually
    // sees on the page.
    await expect(
      anonymousPage.getByRole("heading", { name: projectName }),
    ).toBeVisible();
    await expect(anonymousPage.getByText(monitorName)).toBeVisible();

    await anonymousContext.close();
  });
});
