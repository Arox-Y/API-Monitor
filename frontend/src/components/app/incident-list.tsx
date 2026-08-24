import { ShieldCheck } from "lucide-react";

import { formatDuration, formatRelativeTime } from "@/lib/format";
import type { Incident } from "@/types/api";
import { EmptyState } from "./states";

/**
 * Recent incidents — periods when a monitor was failing.
 *
 * An incident with resolved_at === null is STILL HAPPENING, and that deserves
 * to look different from one that is over. A user scanning this list needs to
 * spot "on fire right now" instantly.
 */
export function IncidentList({
  incidents,
  now,
}: {
  incidents: Incident[];
  now?: Date;
}) {
  if (incidents.length === 0) {
    return (
      <EmptyState
        icon={ShieldCheck}
        title="No incidents"
        description="Nothing has gone down yet. Incidents appear here automatically when a monitor starts failing."
      />
    );
  }

  return (
    <ul className="divide-y divide-border rounded-lg border border-border">
      {incidents.map((incident) => {
        const isOngoing = incident.resolved_at === null;

        return (
          <li key={incident.id} className="flex items-start gap-3 p-3">
            {/* A small bar rather than a dot: it reads as a severity marker
                without needing a legend. Colour is a hint only — the text
                below states the situation explicitly. */}
            <span
              className={`mt-1 h-8 w-0.5 shrink-0 rounded-full ${
                isOngoing ? "bg-status-down" : "bg-border"
              }`}
              aria-hidden="true"
            />

            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-baseline gap-x-2">
                <span className="text-sm font-medium">
                  {incident.monitor_name}
                </span>

                {isOngoing ? (
                  <span className="text-xs font-medium text-status-down">
                    Ongoing
                  </span>
                ) : (
                  <span className="text-xs text-muted-foreground">
                    Resolved · {formatDuration(incident.duration_seconds)}
                  </span>
                )}
              </div>

              <p className="mt-0.5 truncate font-mono text-xs text-muted-foreground">
                {incident.cause}
              </p>
            </div>

            <span className="shrink-0 font-mono text-xs tabular-nums text-muted-foreground">
              {formatRelativeTime(incident.started_at, now)}
            </span>
          </li>
        );
      })}
    </ul>
  );
}
