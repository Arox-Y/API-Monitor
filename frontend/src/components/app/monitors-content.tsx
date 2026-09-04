"use client";

import { Radio } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { StatusBadge } from "@/components/app/status-badge";
import {
  CreateMonitorDialog,
  DeleteMonitorDialog,
} from "@/components/app/monitor-dialogs";
import { EmptyState, ErrorState, TableSkeleton } from "@/components/app/states";
import { useMonitors, useToggleMonitor } from "@/hooks/use-monitors";
import { formatRelativeTime } from "@/lib/format";

/** The monitors list with enable/disable and delete. */
export function MonitorsContent() {
  const { data: monitors, isPending, isError, error, refetch } = useMonitors();
  const toggle = useToggleMonitor();

  if (isPending) {
    return <TableSkeleton rows={4} />;
  }

  if (isError) {
    return (
      <ErrorState
        title="Could not load your monitors"
        description={error.message}
        onRetry={
          <Button variant="outline" onClick={() => refetch()}>
            Try again
          </Button>
        }
      />
    );
  }

  if (monitors.length === 0) {
    // Same reasoning as the projects page: the header's "New monitor" button
    // is already on screen, so the empty state does not repeat it.
    return (
      <EmptyState
        icon={Radio}
        title="No monitors yet"
        description="A monitor checks one URL on a schedule and records whether it responded, how fast, and when it failed. Use New monitor above to create one."
      />
    );
  }

  return (
    <ul className="divide-y divide-border rounded-lg border border-border">
      {monitors.map((monitor) => (
        <li
          key={monitor.id}
          className="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between"
        >
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <span className="font-medium">{monitor.name}</span>
              {monitor.enabled ? (
                <StatusBadge status={monitor.status} />
              ) : (
                <span className="rounded-full border border-border px-2 py-0.5 text-xs text-muted-foreground">
                  Disabled
                </span>
              )}
            </div>

            <p className="mt-1 truncate font-mono text-xs text-muted-foreground">
              {monitor.method} {monitor.url}
            </p>

            <p className="mt-1 font-mono text-xs tabular-nums text-muted-foreground">
              every {monitor.interval_seconds}s · timeout{" "}
              {monitor.timeout_seconds}s · expect {monitor.expected_status} ·
              checked {formatRelativeTime(monitor.last_checked_at)}
            </p>
          </div>

          <div className="flex shrink-0 items-center gap-3">
            <div className="flex items-center gap-2">
              {/* An accessible name is essential: without it a screen reader
                  announces only "switch", with no idea which monitor. */}
              <Switch
                checked={monitor.enabled}
                aria-label={`${monitor.enabled ? "Disable" : "Enable"} ${monitor.name}`}
                disabled={toggle.isPending}
                onCheckedChange={(checked) =>
                  toggle.mutate({ id: monitor.id, enabled: checked })
                }
              />
              <span className="text-xs text-muted-foreground">
                {monitor.enabled ? "On" : "Off"}
              </span>
            </div>

            <DeleteMonitorDialog monitor={monitor} />
          </div>
        </li>
      ))}
    </ul>
  );
}

/** Header action, kept here so the page can stay a Server Component. */
export function MonitorsHeaderAction() {
  return <CreateMonitorDialog />;
}
