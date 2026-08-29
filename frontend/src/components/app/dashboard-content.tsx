"use client";

import Link from "next/link";
import { Radio } from "lucide-react";

import { Button } from "@/components/ui/button";
import { IncidentList } from "@/components/app/incident-list";
import { MonitorTable } from "@/components/app/monitor-table";
import { StatCard } from "@/components/app/stat-card";
import {
  EmptyState,
  ErrorState,
  StatCardSkeleton,
  TableSkeleton,
} from "@/components/app/states";
import { useDashboardStats } from "@/hooks/use-dashboard";
import { useMonitors } from "@/hooks/use-monitors";
import { formatRelativeTime, formatUptime } from "@/lib/format";

/**
 * The dashboard.
 *
 * Counts and uptime come from GET /dashboard/stats, computed in SQL rather
 * than in the browser. That matters: the browser only has the monitors it
 * fetched, and uptime needs to look at every check in 24 hours - potentially
 * thousands of rows nobody wants to send to a phone.
 */
export function DashboardContent() {
  const stats = useDashboardStats();
  const monitors = useMonitors();

  // ---- LOADING ----
  if (stats.isPending || monitors.isPending) {
    return (
      <div className="space-y-8">
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
          <StatCardSkeleton />
          <StatCardSkeleton />
          <StatCardSkeleton />
          <StatCardSkeleton />
        </div>
        <TableSkeleton rows={4} />
      </div>
    );
  }

  // ---- ERROR ----
  if (stats.isError || monitors.isError) {
    const message =
      stats.error?.message ?? monitors.error?.message ?? "Unknown error";
    return (
      <ErrorState
        title="Could not load your dashboard"
        description={message}
        onRetry={
          <Button
            variant="outline"
            onClick={() => {
              stats.refetch();
              monitors.refetch();
            }}
          >
            Try again
          </Button>
        }
      />
    );
  }

  // ---- EMPTY ----
  if (stats.data.total_monitors === 0) {
    return (
      <EmptyState
        icon={Radio}
        title="No monitors yet"
        description="Create a project, then add a monitor to start recording uptime and response times."
        action={<Button render={<Link href="/projects" />}>Go to projects</Button>}
      />
    );
  }

  // ---- SUCCESS ----
  const { total_monitors, monitors_up, monitors_down, uptime_percentage_24h } =
    stats.data;

  return (
    <div className="space-y-8">
      <section aria-labelledby="overview-heading">
        <h2 id="overview-heading" className="sr-only">
          Overview
        </h2>

        <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
          <StatCard label="Total monitors" value={String(total_monitors)} />
          <StatCard label="Up" value={String(monitors_up)} tone="up" />
          <StatCard
            label="Down"
            value={String(monitors_down)}
            tone={monitors_down > 0 ? "down" : "default"}
          />
          <StatCard
            label="Uptime (24h)"
            // The API returns null when no checks exist. That renders as a
            // dash, never "0.00%" - a new account is not a broken account.
            value={formatUptime(uptime_percentage_24h)}
            hint={
              uptime_percentage_24h === null ? "No checks recorded yet" : undefined
            }
          />
        </div>
      </section>

      <section aria-labelledby="monitors-heading" className="space-y-3">
        <h2 id="monitors-heading" className="text-sm font-medium">
          Monitors
        </h2>
        <MonitorTable monitors={monitors.data} />
      </section>

      <div className="grid gap-8 lg:grid-cols-2">
        <section aria-labelledby="incidents-heading" className="space-y-3">
          <h2 id="incidents-heading" className="text-sm font-medium">
            Recent incidents
          </h2>
          <IncidentList incidents={stats.data.recent_incidents} />
        </section>

        <section aria-labelledby="checks-heading" className="space-y-3">
          <h2 id="checks-heading" className="text-sm font-medium">
            Recent checks
          </h2>

          {stats.data.recent_checks.length === 0 ? (
            <div className="rounded-lg border border-dashed border-border px-6 py-8 text-center">
              <p className="text-sm text-muted-foreground">
                No checks yet. The first one runs within a minute.
              </p>
            </div>
          ) : (
            <ul className="divide-y divide-border rounded-lg border border-border">
              {stats.data.recent_checks.map((check) => (
                <li
                  key={check.id}
                  className="flex items-center justify-between gap-3 p-3"
                >
                  <div className="min-w-0">
                    <p className="truncate text-sm">{check.monitor_name}</p>
                    <p className="mt-0.5 font-mono text-xs text-muted-foreground">
                      {/* null status_code means nothing responded at all. */}
                      {check.status_code ?? "no response"}
                      {check.error && ` · ${check.error}`}
                    </p>
                  </div>

                  <div className="shrink-0 text-right">
                    <p
                      className={`font-mono text-xs tabular-nums ${
                        check.success
                          ? "text-muted-foreground"
                          : "text-status-down"
                      }`}
                    >
                      {check.success ? `${check.response_time_ms} ms` : "failed"}
                    </p>
                    <p className="mt-0.5 font-mono text-xs tabular-nums text-muted-foreground">
                      {formatRelativeTime(check.checked_at)}
                    </p>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>
    </div>
  );
}
