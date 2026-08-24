import { IncidentList } from "@/components/app/incident-list";
import { MonitorTable } from "@/components/app/monitor-table";
import { PageHeader } from "@/components/app/page-header";
import { StatCard } from "@/components/app/stat-card";
import { formatRelativeTime, formatUptime } from "@/lib/format";
import { MOCK_DASHBOARD_STATS, MOCK_MONITORS } from "@/lib/mock-data";

/**
 * The dashboard.
 *
 * ⚠️ Currently rendering MOCK data from src/lib/mock-data.ts. On Day 2 this is
 * replaced by a real call to GET /api/v1/dashboard/stats (PLAN.md task 49).
 *
 * Building the screen first, against invented data, is deliberate: it shows us
 * exactly which fields the API must return. Designing the endpoint first and
 * the screen second usually means guessing wrong and reworking both.
 */

// Fixed "now" so the mock relative times ("2m ago") stay stable. Once real
// data arrives this goes away and the actual current time is used.
const MOCK_NOW = new Date("2026-08-24T10:30:00Z");

export default function DashboardPage() {
  const stats = MOCK_DASHBOARD_STATS;

  return (
    <div>
      <PageHeader
        title="Dashboard"
        description="Health of every monitor across your projects."
      />

      <div className="space-y-8 px-4 py-6 lg:px-6">
        {/* Honest banner. CLAUDE.md forbids presenting placeholder data as if
            it were real — so we say so on screen, not just in a comment. */}
        <div className="rounded-md border border-status-degraded/30 bg-status-degraded/10 px-3 py-2 text-xs text-muted-foreground">
          <span className="font-medium text-foreground">Sample data.</span>{" "}
          The backend is not connected yet — these numbers are placeholders.
        </div>

        {/* ---- Stats ---- */}
        <section aria-labelledby="overview-heading">
          <h2 id="overview-heading" className="sr-only">
            Overview
          </h2>

          <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
            <StatCard
              label="Total monitors"
              value={String(stats.total_monitors)}
            />
            <StatCard
              label="Up"
              value={String(stats.monitors_up)}
              tone="up"
            />
            <StatCard
              label="Down"
              value={String(stats.monitors_down)}
              tone={stats.monitors_down > 0 ? "down" : "default"}
            />
            <StatCard
              label="Uptime (24h)"
              // formatUptime renders null as "—", never "0.00%".
              // "we have not measured" and "everything failed" are different
              // facts and must never look the same.
              value={formatUptime(stats.uptime_percentage_24h)}
              hint={
                stats.uptime_percentage_24h === null
                  ? "No checks recorded yet"
                  : undefined
              }
            />
          </div>
        </section>

        {/* ---- Monitors ---- */}
        <section aria-labelledby="monitors-heading" className="space-y-3">
          <h2 id="monitors-heading" className="text-sm font-medium">
            Monitors
          </h2>
          <MonitorTable monitors={MOCK_MONITORS} now={MOCK_NOW} />
        </section>

        {/* ---- Incidents + recent checks ---- */}
        <div className="grid gap-8 lg:grid-cols-2">
          <section aria-labelledby="incidents-heading" className="space-y-3">
            <h2 id="incidents-heading" className="text-sm font-medium">
              Recent incidents
            </h2>
            <IncidentList
              incidents={stats.recent_incidents}
              now={MOCK_NOW}
            />
          </section>

          <section aria-labelledby="checks-heading" className="space-y-3">
            <h2 id="checks-heading" className="text-sm font-medium">
              Recent checks
            </h2>

            <ul className="divide-y divide-border rounded-lg border border-border">
              {stats.recent_checks.map((check) => (
                <li
                  key={check.id}
                  className="flex items-center justify-between gap-3 p-3"
                >
                  <div className="min-w-0">
                    <p className="truncate text-sm">{check.monitor_name}</p>
                    <p className="mt-0.5 font-mono text-xs text-muted-foreground">
                      {/* status_code is null when nothing responded at all.
                          Showing "—" is honest; showing 0 would be a lie. */}
                      {check.status_code ?? "no response"}
                      {check.error && ` · ${check.error}`}
                    </p>
                  </div>

                  <div className="shrink-0 text-right">
                    <p
                      className={`font-mono text-xs tabular-nums ${
                        check.success ? "text-muted-foreground" : "text-status-down"
                      }`}
                    >
                      {check.success ? `${check.response_time_ms} ms` : "failed"}
                    </p>
                    <p className="mt-0.5 font-mono text-xs tabular-nums text-muted-foreground">
                      {formatRelativeTime(check.checked_at, MOCK_NOW)}
                    </p>
                  </div>
                </li>
              ))}
            </ul>
          </section>
        </div>
      </div>
    </div>
  );
}
