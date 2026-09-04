import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import path from "node:path";

/**
 * Vitest configuration.
 *
 * Vitest rather than Jest because this project already uses Vite-style tooling
 * through Next.js: it needs far less configuration to understand TypeScript
 * and ESM, and it runs noticeably faster.
 */
export default defineConfig({
  plugins: [react()],
  test: {
    // jsdom is a fake browser implemented in JavaScript. Component tests need
    // a DOM to render into, and jsdom provides one without launching Chrome -
    // which is what makes these tests take milliseconds instead of seconds.
    environment: "jsdom",

    // Runs before every test file: adds the extra matchers from jest-dom, such
    // as toBeInTheDocument().
    setupFiles: ["./vitest.setup.ts"],

    globals: true,

    // Run tests in WORKER THREADS rather than forked child processes.
    //
    // Vitest defaults to `forks`, which spawns a full Node process per worker.
    // On this machine that timed out with "Failed to start forks worker" -
    // the project lives inside OneDrive, and process startup there is slow
    // enough to exceed the default timeout.
    //
    // Threads start in milliseconds instead. The trade-off is weaker isolation
    // between test files, which matters only for tests that mutate global
    // state - none of ours do.
    pool: "threads",

    // One file at a time. Parallelism buys little for a suite this small and
    // removes a whole class of flakiness.
    //
    // Note: Vitest 4 REMOVED `poolOptions` - those settings are top-level now.
    // Using the old shape prints a deprecation warning and is ignored.
    fileParallelism: false,

    // Generous, because OneDrive makes file access unpredictable here.
    testTimeout: 15_000,
    hookTimeout: 15_000,

    // Only OUR tests. Without this exclusion Vitest would walk node_modules
    // and try to run other packages' test files.
    include: ["src/**/*.{test,spec}.{ts,tsx}"],
    exclude: ["node_modules", ".next", "e2e"],
  },
  resolve: {
    alias: {
      // The same @/* alias the app uses, so tests import exactly like the
      // components they are testing.
      "@": path.resolve(__dirname, "./src"),
    },
  },
});
