/**
 * Small formatting helpers.
 *
 * These are "pure functions": same input always gives the same output, and
 * they touch nothing outside themselves. That makes them the easiest kind of
 * code to test — no browser, no server, no database required.
 */

/**
 * Turn a number of seconds into something readable: "27m", "2h 15m", "45s".
 * Returns "—" when the value is null (an incident that is still open).
 */
export function formatDuration(seconds: number | null): string {
  if (seconds === null) return "—";
  if (seconds < 60) return `${seconds}s`;

  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m`;

  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  if (hours < 24) {
    return remainingMinutes === 0 ? `${hours}h` : `${hours}h ${remainingMinutes}m`;
  }

  const days = Math.floor(hours / 24);
  const remainingHours = hours % 24;
  return remainingHours === 0 ? `${days}d` : `${days}d ${remainingHours}h`;
}

/**
 * Turn a timestamp into "just now", "5m ago", "3h ago", "2d ago".
 *
 * `now` is a parameter rather than being read inside the function so that a
 * test can pass a fixed time. A function that secretly calls new Date() gives
 * a different answer every run, which is impossible to test reliably.
 */
export function formatRelativeTime(
  isoTimestamp: string | null,
  now: Date = new Date(),
): string {
  if (isoTimestamp === null) return "never";

  const then = new Date(isoTimestamp);
  const seconds = Math.floor((now.getTime() - then.getTime()) / 1000);

  if (seconds < 0) return "just now"; // clock skew; do not show "-3s ago"
  if (seconds < 45) return "just now";
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
  if (seconds < 86_400) return `${Math.floor(seconds / 3600)}h ago`;
  return `${Math.floor(seconds / 86_400)}d ago`;
}

/**
 * Format an uptime percentage.
 *
 * null means "we have no checks yet" and MUST render as "—", never "0.0%".
 * Showing unknown as zero would tell the user everything is broken when in
 * fact nothing has been measured. See docs/FRONTEND_DESIGN.md §4.
 */
export function formatUptime(percentage: number | null): string {
  if (percentage === null) return "—";
  return `${percentage.toFixed(2)}%`;
}

/** Format a response time. Checks that never responded have no useful time. */
export function formatResponseTime(
  milliseconds: number,
  succeeded: boolean,
): string {
  if (!succeeded) return "—";
  return `${milliseconds} ms`;
}

/** Format an HTTP status code; null means no response ever arrived. */
export function formatStatusCode(code: number | null): string {
  return code === null ? "—" : String(code);
}
