/**
 * TypeScript shapes for everything the API returns.
 *
 * These mirror docs/API_SPEC.md exactly. If the backend changes, this file
 * changes with it — it is the single place the frontend describes API data.
 *
 * Note the snake_case field names (`response_time_ms`, not `responseTimeMs`).
 * That is deliberate: the Go backend and the database both use snake_case, so
 * using it here too means zero renaming as data moves between the three.
 */

/** A monitor's current state. `pending` means it has never been checked yet. */
export type MonitorStatus = "pending" | "up" | "down";

/** Overall state of a project on its public status page. */
export type OverallStatus = "up" | "degraded" | "down";

/** HTTP methods a monitor may use. Matches the database CHECK constraint. */
export type HttpMethod = "GET" | "POST" | "HEAD";

export type Project = {
  id: string;
  name: string;
  description: string;
  slug: string;
  is_public: boolean;
  monitor_count: number;
  created_at: string;
  updated_at: string;
};

export type Monitor = {
  id: string;
  project_id: string;
  name: string;
  url: string;
  method: HttpMethod;
  interval_seconds: number;
  timeout_seconds: number;
  expected_status: number;
  enabled: boolean;
  status: MonitorStatus;
  /** null when the monitor has never been checked. */
  last_checked_at: string | null;
  created_at: string;
  updated_at: string;
};

export type Check = {
  id: string;
  checked_at: string;
  success: boolean;
  /**
   * null means no HTTP response arrived at all — a timeout, a DNS failure, or
   * a refused connection. It is NOT the same as 0, and definitely not 200.
   */
  status_code: number | null;
  response_time_ms: number;
  error: string;
};

/** A check shown on the dashboard, where we also need to say which monitor. */
export type CheckWithMonitor = Check & { monitor_name: string };

export type Incident = {
  id: string;
  monitor_id: string;
  monitor_name: string;
  started_at: string;
  /** null means the incident is still open — the monitor is still down. */
  resolved_at: string | null;
  /** null while the incident is still open. */
  duration_seconds: number | null;
  cause: string;
};

/**
 * Everything the dashboard screen needs, in one response.
 * See ADR-005 for why this is one aggregate endpoint rather than five.
 */
export type DashboardStats = {
  total_monitors: number;
  monitors_up: number;
  monitors_down: number;
  /**
   * null when the user has no checks yet.
   *
   * This is important: null ("we do not know") and 0 ("everything failed") are
   * completely different facts, and the UI must never show one as the other.
   */
  uptime_percentage_24h: number | null;
  recent_incidents: Incident[];
  recent_checks: CheckWithMonitor[];
};

/**
 * Everything the monitor detail screen needs, from one request.
 * Returned by GET /api/v1/monitors/{id}/checks.
 */
export type MonitorDetail = {
  monitor: Monitor;
  checks: Check[];
  incidents: Incident[];
  /** null when the monitor has no checks in the window. Not the same as 0. */
  uptime_percentage_24h: number | null;
};
