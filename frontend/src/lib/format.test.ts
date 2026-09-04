import { describe, expect, it } from "vitest";

import {
  formatDuration,
  formatRelativeTime,
  formatResponseTime,
  formatStatusCode,
  formatUptime,
} from "./format";

/**
 * Tests for the formatting helpers.
 *
 * These are pure functions - same input, same output, no browser, no network -
 * which makes them the cheapest possible tests to write and run. They are also
 * where a surprising number of user-visible bugs live, because "display a
 * number" hides a lot of edge cases.
 */

describe("formatUptime", () => {
  it("formats a percentage to two decimal places", () => {
    expect(formatUptime(99.4235)).toBe("99.42%");
    expect(formatUptime(100)).toBe("100.00%");
  });

  // THE MOST IMPORTANT TEST IN THIS FILE.
  it("renders null as a dash, NEVER as 0%", () => {
    // null means "we have not measured anything yet".
    // 0% means "every single check failed".
    //
    // These are completely different facts. Showing a brand-new account
    // "0.00% uptime" tells them their service is dead when nothing has been
    // checked at all.
    expect(formatUptime(null)).toBe("—");
    expect(formatUptime(null)).not.toBe("0.00%");
  });

  it("still shows a real zero as 0.00%", () => {
    // The flip side: an account whose checks genuinely all failed must see
    // 0.00%, not a dash. Both facts are real and must look different.
    expect(formatUptime(0)).toBe("0.00%");
  });
});

describe("formatDuration", () => {
  it("formats seconds, minutes, hours and days", () => {
    expect(formatDuration(45)).toBe("45s");
    expect(formatDuration(60)).toBe("1m");
    expect(formatDuration(1620)).toBe("27m");
    expect(formatDuration(3600)).toBe("1h");
    expect(formatDuration(3900)).toBe("1h 5m");
    expect(formatDuration(86400)).toBe("1d");
    expect(formatDuration(90000)).toBe("1d 1h");
  });

  it("renders null as a dash", () => {
    // null here means the incident is STILL OPEN, so it has no duration yet.
    expect(formatDuration(null)).toBe("—");
  });

  it("handles the boundaries between units", () => {
    // Off-by-one errors live exactly here.
    expect(formatDuration(59)).toBe("59s");
    expect(formatDuration(3599)).toBe("59m");
  });
});

describe("formatRelativeTime", () => {
  // `now` is passed IN rather than read inside the function. A function that
  // secretly calls new Date() gives a different answer every run and cannot be
  // tested reliably.
  const now = new Date("2026-08-30T12:00:00Z");

  it("describes recent times", () => {
    expect(formatRelativeTime("2026-08-30T11:59:30Z", now)).toBe("just now");
    expect(formatRelativeTime("2026-08-30T11:55:00Z", now)).toBe("5m ago");
    expect(formatRelativeTime("2026-08-30T09:00:00Z", now)).toBe("3h ago");
    expect(formatRelativeTime("2026-08-28T12:00:00Z", now)).toBe("2d ago");
  });

  it("says 'never' when there is no timestamp", () => {
    // A monitor that has never been checked.
    expect(formatRelativeTime(null, now)).toBe("never");
  });

  it("does not produce a negative time when clocks disagree", () => {
    // The server and the browser can disagree by a few seconds. Without a
    // guard this would render "-3s ago", which looks broken.
    const future = "2026-08-30T12:00:30Z";
    expect(formatRelativeTime(future, now)).toBe("just now");
  });
});

describe("formatStatusCode", () => {
  it("shows the code when there is one", () => {
    expect(formatStatusCode(200)).toBe("200");
    expect(formatStatusCode(503)).toBe("503");
  });

  it("renders null as a dash", () => {
    // null means NO HTTP RESPONSE ARRIVED - a timeout, a DNS failure, a
    // refused connection. There is no status code to show.
    expect(formatStatusCode(null)).toBe("—");
    // Showing 0 would invent a status code that never existed.
    expect(formatStatusCode(null)).not.toBe("0");
  });
});

describe("formatResponseTime", () => {
  it("shows milliseconds for a successful check", () => {
    expect(formatResponseTime(143, true)).toBe("143 ms");
  });

  it("shows a dash for a failed check", () => {
    // A failed check's elapsed time is how long we waited before giving up,
    // not how fast the service responded. Showing it as a response time would
    // be misleading - a 10-second timeout is not "a 10-second response".
    expect(formatResponseTime(10000, false)).toBe("—");
  });
});
