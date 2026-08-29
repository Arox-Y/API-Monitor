"use client";

import Link from "next/link";
import { ArrowLeft } from "lucide-react";

import { Button } from "@/components/ui/button";
import { CheckHistory } from "@/components/app/check-history";
import { IncidentList } from "@/components/app/incident-list";
import { ResponseTimeChart } from "@/components/app/response-time-chart";
import { StatCard } from "@/components/app/stat-card";
import { StatusBadge } from "@/components/app/status-badge";
import { ErrorState, StatCardSkeleton, TableSkeleton } from "@/components/app/states";
import { useMonitorDetail } from "@/hooks/use-dashboard";
import { formatRelativeTime, formatUptime } from "@/lib/format";

/** One monitor: current state, response-time chart, incidents, raw history. */
export function MonitorDetailContent({ monitorId }: { monitorId: string }) {
  const { data, isPending, isError, error, refetch } = useMonitorDetail(monitorId);

  if (isPending) {
    return (
      <div className="space-y-8">
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
          <StatCardSkeleton />
          <StatCardSkeleton />
          <StatCardSkeleton />
          <StatCardSkeleton />
        </div>
        <TableSkeleton rows={5} />
      </div>
    );
  }

  if (isError) {
    // A 404 here means the monitor does not exist OR is not yours - the API
    // deliberately cannot tell those apart, so neither can this message.
    const notFound = error.message.toLowerCase().includes("not found");

    return (
      <ErrorState
        title={notFound ? "Monitor not found" : "Could not load this monitor"}
        description={
          notFound
            ? "It may have been deleted, or the link may be wrong."
            : error.message
        }
        onRetry={
          notFound ? (
            <Button variant="outline" render={<Link href="/monitors" />}>
              Back to monitors
            </Button>
          ) : (
            <Button variant="outline" onClick={() => refetch()}>
              Try again
            </Button>
          )
        }
      />
    );
  }

  const { monitor, checks, incidents, uptime_percentage_24h } = data;

  const successful = checks.filter((c) => c.success);
  const averageMs =
    successful.length === 0
      ? null
      : Math.round(
          successful.reduce((sum, c) => sum + c.response_time_ms, 0) /
            successful.length,
        );

  return (
    <div className="space-y-8">
      <section aria-labelledby="summary-heading">
        <h2 id="summary-heading" className="sr-only">
          Summary
        </h2>

        <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
          <div className="rounded-lg border border-border p-4">
            <p className="text-sm text-muted-foreground">Status</p>
            <div className="mt-2">
              {monitor.enabled ? (
                <StatusBadge status={monitor.status} />
              ) : (
                <span className="rounded-full border border-border px-2 py-0.5 text-xs text-muted-foreground">
                  Disabled
                </span>
              )}
            </div>
          </div>

          <StatCard
            label="Uptime (24h)"
            // null renders as a dash. "Not measured" and "everything failed"
            // must never look the same.
            value={formatUptime(uptime_percentage_24h)}
            hint={uptime_percentage_24h === null ? "No checks yet" : undefined}
          />

          <StatCard
            label="Avg response"
            value={averageMs === null ? "—" : `${averageMs} ms`}
            hint={averageMs === null ? "No successful checks" : undefined}
          />

          <StatCard label="Checks (24h)" value={String(checks.length)} />
        </div>
      </section>

      {/* Configuration, so you can see what is actually being checked. */}
      <section aria-labelledby="config-heading" className="space-y-3">
        <h2 id="config-heading" className="text-sm font-medium">
          Configuration
        </h2>
        <dl className="grid gap-x-6 gap-y-2 rounded-lg border border-border p-4 text-sm sm:grid-cols-2">
          <Row label="URL" value={`${monitor.method} ${monitor.url}`} />
          <Row label="Interval" value={`every ${monitor.interval_seconds}s`} />
          <Row label="Timeout" value={`${monitor.timeout_seconds}s`} />
          <Row label="Expected status" value={String(monitor.expected_status)} />
          <Row
            label="Last checked"
            value={formatRelativeTime(monitor.last_checked_at)}
          />
          <Row label="Enabled" value={monitor.enabled ? "Yes" : "No"} />
        </dl>
      </section>

      <ResponseTimeChart checks={checks} />

      <section aria-labelledby="incidents-heading" className="space-y-3">
        <h2 id="incidents-heading" className="text-sm font-medium">
          Incidents
        </h2>
        <IncidentList incidents={incidents} />
      </section>

      <section aria-labelledby="history-heading" className="space-y-3">
        <h2 id="history-heading" className="text-sm font-medium">
          Check history
        </h2>
        <CheckHistory checks={checks} />
      </section>
    </div>
  );
}

/** One label/value pair in the configuration list. */
function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-4 sm:block">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="truncate font-mono text-xs tabular-nums sm:mt-0.5">
        {value}
      </dd>
    </div>
  );
}

/** Back link, kept here so the page can stay a Server Component. */
export function BackToMonitors() {
  return (
    <Button variant="outline" size="sm" render={<Link href="/monitors" />}>
      <ArrowLeft className="size-4" aria-hidden="true" />
      Monitors
    </Button>
  );
}
