import Link from "next/link";
import { Radio } from "lucide-react";

import { formatRelativeTime } from "@/lib/format";
import type { Monitor } from "@/types/api";
import { EmptyState } from "./states";
import { StatusBadge } from "./status-badge";

/**
 * The list of monitors.
 *
 * On desktop this is a real <table>. On mobile it becomes stacked cards,
 * because a 5-column table on a 375px screen either overflows sideways or
 * squashes into unreadable slivers. Two layouts, one set of data.
 *
 * Note it handles its own empty state — a component that renders a table with
 * no rows and no explanation looks broken.
 */
export function MonitorTable({
  monitors,
  now,
}: {
  monitors: Monitor[];
  /** Passed in rather than read inside, so output is predictable and testable. */
  now?: Date;
}) {
  if (monitors.length === 0) {
    return (
      <EmptyState
        icon={Radio}
        title="No monitors yet"
        description="Monitors check your endpoints on a schedule and record whether they responded, how fast, and when they failed."
      />
    );
  }

  return (
    <>
      {/* ---------- Desktop table ---------- */}
      <div className="hidden overflow-hidden rounded-lg border border-border md:block">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border bg-muted/40">
              {/*
                scope="col" tells assistive technology that these cells label
                their whole column. Without it a screen reader reads a table as
                an unlabelled grid of values, which is nearly useless.
              */}
              <th scope="col" className="px-4 py-2.5 text-left font-medium text-muted-foreground">
                Monitor
              </th>
              <th scope="col" className="px-4 py-2.5 text-left font-medium text-muted-foreground">
                Status
              </th>
              <th scope="col" className="px-4 py-2.5 text-left font-medium text-muted-foreground">
                Interval
              </th>
              <th scope="col" className="px-4 py-2.5 text-left font-medium text-muted-foreground">
                Last check
              </th>
            </tr>
          </thead>

          <tbody className="divide-y divide-border">
            {monitors.map((monitor) => (
              <tr key={monitor.id} className="transition-colors hover:bg-muted/30">
                <td className="px-4 py-3">
                  <Link
                    href={`/monitors/${monitor.id}`}
                    className="font-medium hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
                  >
                    {monitor.name}
                  </Link>
                  {/* Monospace for machine data: URLs, codes, timings. It scans
                      better and signals "this is data, not prose". */}
                  <p className="mt-0.5 truncate font-mono text-xs text-muted-foreground">
                    {monitor.method} {monitor.url}
                  </p>
                </td>

                <td className="px-4 py-3">
                  {monitor.enabled ? (
                    <StatusBadge status={monitor.status} />
                  ) : (
                    <span className="text-xs text-muted-foreground">Disabled</span>
                  )}
                </td>

                <td className="px-4 py-3 font-mono text-xs tabular-nums text-muted-foreground">
                  {monitor.interval_seconds}s
                </td>

                <td className="px-4 py-3 font-mono text-xs tabular-nums text-muted-foreground">
                  {formatRelativeTime(monitor.last_checked_at, now)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* ---------- Mobile cards ---------- */}
      <ul className="divide-y divide-border rounded-lg border border-border md:hidden">
        {monitors.map((monitor) => (
          <li key={monitor.id} className="p-3">
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <Link
                  href={`/monitors/${monitor.id}`}
                  className="text-sm font-medium hover:underline"
                >
                  {monitor.name}
                </Link>
                <p className="mt-0.5 truncate font-mono text-xs text-muted-foreground">
                  {monitor.url}
                </p>
              </div>

              <div className="shrink-0">
                {monitor.enabled ? (
                  <StatusBadge status={monitor.status} />
                ) : (
                  <span className="text-xs text-muted-foreground">Disabled</span>
                )}
              </div>
            </div>

            <p className="mt-2 font-mono text-xs tabular-nums text-muted-foreground">
              every {monitor.interval_seconds}s · checked{" "}
              {formatRelativeTime(monitor.last_checked_at, now)}
            </p>
          </li>
        ))}
      </ul>
    </>
  );
}
