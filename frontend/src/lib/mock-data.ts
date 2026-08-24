/**
 * ⚠️ MOCK DATA — NOT REAL ⚠️
 *
 * Hard-coded sample data so the dashboard can be built and reviewed before the
 * API exists. Every value here is invented.
 *
 * This entire file is DELETED on Day 2, when the real API endpoints are wired
 * up (PLAN.md task 49). If you can still see this file after Day 2, something
 * was left half-finished.
 *
 * It is typed with the real API types on purpose: if the mock shape does not
 * match what the backend will send, TypeScript fails the build now rather than
 * letting us discover it later.
 */

import type {
  CheckWithMonitor,
  DashboardStats,
  Incident,
  Monitor,
} from "@/types/api";

/** Fixed reference time so the mock data never shifts between renders. */
const NOW = new Date("2026-08-24T10:30:00Z");

/** Helper: an ISO timestamp N minutes before NOW. */
function minutesAgo(minutes: number): string {
  return new Date(NOW.getTime() - minutes * 60_000).toISOString();
}

export const MOCK_MONITORS: Monitor[] = [
  {
    id: "7a2d0001-0000-4000-8000-000000000001",
    project_id: "3c1b0001-0000-4000-8000-000000000001",
    name: "Checkout API",
    url: "https://api.acme.com/health",
    method: "GET",
    interval_seconds: 60,
    timeout_seconds: 10,
    expected_status: 200,
    enabled: true,
    status: "up",
    last_checked_at: minutesAgo(1),
    created_at: minutesAgo(20_000),
    updated_at: minutesAgo(20_000),
  },
  {
    id: "7a2d0001-0000-4000-8000-000000000002",
    project_id: "3c1b0001-0000-4000-8000-000000000001",
    name: "Auth Service",
    url: "https://auth.acme.com/healthz",
    method: "GET",
    interval_seconds: 60,
    timeout_seconds: 10,
    expected_status: 200,
    enabled: true,
    status: "up",
    last_checked_at: minutesAgo(1),
    created_at: minutesAgo(19_000),
    updated_at: minutesAgo(19_000),
  },
  {
    id: "7a2d0001-0000-4000-8000-000000000003",
    project_id: "3c1b0001-0000-4000-8000-000000000002",
    name: "Search API",
    url: "https://search.acme.com/ping",
    method: "GET",
    interval_seconds: 30,
    timeout_seconds: 5,
    expected_status: 200,
    enabled: true,
    status: "down",
    last_checked_at: minutesAgo(1),
    created_at: minutesAgo(15_000),
    updated_at: minutesAgo(15_000),
  },
  {
    id: "7a2d0001-0000-4000-8000-000000000004",
    project_id: "3c1b0001-0000-4000-8000-000000000002",
    name: "Webhook Receiver",
    url: "https://hooks.acme.com/status",
    method: "POST",
    interval_seconds: 300,
    timeout_seconds: 15,
    expected_status: 202,
    enabled: true,
    status: "up",
    last_checked_at: minutesAgo(3),
    created_at: minutesAgo(9_000),
    updated_at: minutesAgo(9_000),
  },
  {
    id: "7a2d0001-0000-4000-8000-000000000005",
    project_id: "3c1b0001-0000-4000-8000-000000000003",
    name: "Marketing Site",
    url: "https://acme.com",
    method: "HEAD",
    interval_seconds: 300,
    timeout_seconds: 10,
    expected_status: 200,
    enabled: false,
    status: "pending",
    last_checked_at: null,
    created_at: minutesAgo(120),
    updated_at: minutesAgo(120),
  },
];

export const MOCK_INCIDENTS: Incident[] = [
  {
    id: "b4000001-0000-4000-8000-000000000001",
    monitor_id: "7a2d0001-0000-4000-8000-000000000003",
    monitor_name: "Search API",
    started_at: minutesAgo(75),
    resolved_at: null, // still open — the site is down right now
    duration_seconds: null,
    cause: "expected status 200, got 503",
  },
  {
    id: "b4000001-0000-4000-8000-000000000002",
    monitor_id: "7a2d0001-0000-4000-8000-000000000001",
    monitor_name: "Checkout API",
    started_at: minutesAgo(320),
    resolved_at: minutesAgo(293),
    duration_seconds: 1_620,
    cause: "context deadline exceeded",
  },
  {
    id: "b4000001-0000-4000-8000-000000000003",
    monitor_id: "7a2d0001-0000-4000-8000-000000000002",
    monitor_name: "Auth Service",
    started_at: minutesAgo(1_450),
    resolved_at: minutesAgo(1_444),
    duration_seconds: 360,
    cause: "expected status 200, got 502",
  },
];

export const MOCK_RECENT_CHECKS: CheckWithMonitor[] = [
  {
    id: "e1000001-0000-4000-8000-000000000001",
    monitor_name: "Search API",
    checked_at: minutesAgo(1),
    success: false,
    status_code: 503,
    response_time_ms: 89,
    error: "expected status 200, got 503",
  },
  {
    id: "e1000001-0000-4000-8000-000000000002",
    monitor_name: "Checkout API",
    checked_at: minutesAgo(1),
    success: true,
    status_code: 200,
    response_time_ms: 143,
    error: "",
  },
  {
    id: "e1000001-0000-4000-8000-000000000003",
    monitor_name: "Auth Service",
    checked_at: minutesAgo(2),
    success: true,
    status_code: 200,
    response_time_ms: 91,
    error: "",
  },
  {
    id: "e1000001-0000-4000-8000-000000000004",
    monitor_name: "Webhook Receiver",
    checked_at: minutesAgo(3),
    success: true,
    status_code: 202,
    response_time_ms: 212,
    error: "",
  },
  {
    id: "e1000001-0000-4000-8000-000000000005",
    monitor_name: "Search API",
    // A timeout: no response ever arrived, so there is no status code.
    checked_at: minutesAgo(4),
    success: false,
    status_code: null,
    response_time_ms: 5_000,
    error: "context deadline exceeded",
  },
];

export const MOCK_DASHBOARD_STATS: DashboardStats = {
  total_monitors: 5,
  monitors_up: 3,
  monitors_down: 1,
  uptime_percentage_24h: 99.42,
  recent_incidents: MOCK_INCIDENTS,
  recent_checks: MOCK_RECENT_CHECKS,
};

/**
 * An account with nothing in it yet, so the empty states can be checked.
 * Note uptime is null, not 0 — we have no data, which is not the same as
 * "everything failed".
 */
export const MOCK_EMPTY_STATS: DashboardStats = {
  total_monitors: 0,
  monitors_up: 0,
  monitors_down: 0,
  uptime_percentage_24h: null,
  recent_incidents: [],
  recent_checks: [],
};
