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
import { useMonitors } from "@/hooks/use-monitors";
import { formatUptime } from "@/lib/format";

/**
 * The dashboard, reading LIVE data from the API.
 *
 * A Client Component, because it uses TanStack Query hooks. The page and
 * layout around it stay server-rendered - the client boundary is kept as
 * small as the interactivity requires.
 *
 * This replaces the mock data used on Day 1.
 */
export function DashboardContent() {
  // isPending / isError / data come from the library. Handling all four
  // states is cheap precisely because we are not tracking them by hand.
  const { data: monitors, isPending, isError, error, refetch } = useMonitors();

  // ---- LOADING ----
  if (isPending) {
    return (
      <div className="space-y-8">
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
          {/* Skeletons match the real card dimensions, so nothing shifts when
              the data arrives. A skeleton of the wrong size causes the very
              jump it was meant to prevent. */}
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
  if (isError) {
    return (
      <ErrorState
        title="Could not load your monitors"
        description={
          error.message || "Something went wrong while talking to the server."
        }
        onRetry={
          <Button variant="outline" onClick={() => refetch()}>
            Try again
          </Button>
        }
      />
    );
  }

  // ---- EMPTY ----
  if (monitors.length === 0) {
    return (
      <EmptyState
        icon={Radio}
        title="No monitors yet"
        description="Create a project, then add a monitor to start recording uptime and response times."
        action={
          <Button render={<Link href="/projects" />}>Go to projects</Button>
        }
      />
    );
  }

  // ---- SUCCESS ----
  const up = monitors.filter((m) => m.status === "up").length;
  const down = monitors.filter((m) => m.status === "down").length;

  // Every monitor is 'pending' until the worker runs, which is Day 3. Until
  // then there are no checks, so uptime is genuinely UNKNOWN.
  //
  // null renders as an em dash, never "0.00%". Showing unknown as zero would
  // tell the user everything is broken when nothing has been measured yet.
  const checked = monitors.filter((m) => m.status !== "pending").length;
  const uptime = checked === 0 ? null : (up / checked) * 100;

  return (
    <div className="space-y-8">
      <section aria-labelledby="overview-heading">
        <h2 id="overview-heading" className="sr-only">
          Overview
        </h2>

        <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
          <StatCard label="Total monitors" value={String(monitors.length)} />
          <StatCard label="Up" value={String(up)} tone="up" />
          <StatCard
            label="Down"
            value={String(down)}
            tone={down > 0 ? "down" : "default"}
          />
          <StatCard
            label="Uptime (24h)"
            value={formatUptime(uptime)}
            hint={uptime === null ? "No checks recorded yet" : undefined}
          />
        </div>
      </section>

      <section aria-labelledby="monitors-heading" className="space-y-3">
        <h2 id="monitors-heading" className="text-sm font-medium">
          Monitors
        </h2>
        <MonitorTable monitors={monitors} />
      </section>

      <section aria-labelledby="incidents-heading" className="space-y-3">
        <h2 id="incidents-heading" className="text-sm font-medium">
          Recent incidents
        </h2>
        {/* Incidents are created by the monitoring worker, which is built on
            Day 3. Until then this is genuinely empty, and says so rather than
            pretending. */}
        <IncidentList incidents={[]} />
      </section>
    </div>
  );
}
