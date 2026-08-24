import {
  Bell,
  ChevronDown,
  ChevronRight,
  CircleCheck,
  CircleDashed,
  CircleX,
  Expand,
  FolderKanban,
  LayoutDashboard,
  Radio,
  Search,
  Settings,
  Sparkle,
  Star,
} from "lucide-react";

/**
 * Floating product preview — a static mockup of the dashboard, shown under the
 * hero and clipped at the bottom to invite scrolling.
 *
 * Server Component. Everything here is presentational markup; the real
 * dashboard is built separately behind /login.
 *
 * The mockup intentionally previews the actual product surface (monitors,
 * status, response times, incidents) rather than generic filler rows.
 */

type MonitorStatus = "up" | "down" | "pending";

const MONITORS: ReadonlyArray<{
  name: string;
  project: string;
  status: MonitorStatus;
  response: string;
  lastCheck: string;
}> = [
  { name: "Checkout API", project: "Acme Payments", status: "up", response: "143 ms", lastCheck: "12s ago" },
  { name: "Auth Service", project: "Acme Core", status: "up", response: "89 ms", lastCheck: "27s ago" },
  { name: "Search API", project: "Acme Core", status: "down", response: "—", lastCheck: "8s ago" },
  { name: "Webhooks", project: "Acme Payments", status: "up", response: "212 ms", lastCheck: "41s ago" },
  { name: "CDN Edge", project: "Marketing", status: "pending", response: "—", lastCheck: "never" },
  { name: "Billing API", project: "Acme Payments", status: "up", response: "176 ms", lastCheck: "19s ago" },
] as const;

const FEED: ReadonlyArray<{ initials: string; tint: string; text: string; time: string }> = [
  { initials: "SA", tint: "bg-rose-500/20 text-rose-300", text: "Search API went down", time: "8s" },
  { initials: "CA", tint: "bg-emerald-500/20 text-emerald-300", text: "Checkout API recovered", time: "4m" },
  { initials: "WH", tint: "bg-sky-500/20 text-sky-300", text: "Webhooks latency normal", time: "12m" },
  { initials: "BA", tint: "bg-indigo-500/20 text-indigo-300", text: "Billing API deployed", time: "1h" },
] as const;

/** Status pill: icon + colour + text, never colour alone (a11y). */
function StatusPill({ status }: { status: MonitorStatus }) {
  const map = {
    up: {
      Icon: CircleCheck,
      label: "Up",
      className: "border-emerald-500/25 bg-emerald-500/10 text-emerald-300",
    },
    down: {
      Icon: CircleX,
      label: "Down",
      className: "border-rose-500/25 bg-rose-500/10 text-rose-300",
    },
    pending: {
      Icon: CircleDashed,
      label: "Pending",
      className: "border-slate-500/25 bg-slate-500/10 text-slate-400",
    },
  } as const;

  const { Icon, label, className } = map[status];

  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-[11px] font-medium ${className}`}
    >
      <Icon className="size-3" aria-hidden="true" />
      {label}
    </span>
  );
}

function SidebarItem({
  icon: Icon,
  label,
  active = false,
}: {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  active?: boolean;
}) {
  return (
    <div
      className={`flex items-center gap-2 rounded-md px-2 py-1.5 text-[12px] ${
        active ? "bg-white/[0.07] text-white" : "text-slate-400"
      }`}
    >
      <Icon className="size-3.5" />
      {label}
    </div>
  );
}

export function ProductPreview() {
  return (
    <section
      id="preview"
      className="relative mx-auto max-w-6xl px-4 pb-0 sm:px-6 lg:px-8"
    >
      {/* Rim lighting radiating from the top edge of the window. */}
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-x-8 -top-8 h-40 blur-2xl"
        style={{
          background:
            "radial-gradient(ellipse at 50% 0%, rgba(56,189,248,0.35) 0%, rgba(96,165,250,0.18) 45%, transparent 72%)",
        }}
      />

      {/* Clip the window so it runs off the bottom of the fold. */}
      <div className="relative max-h-[560px] overflow-hidden">
        <div className="relative rounded-2xl border border-white/10 bg-space-700/90 shadow-2xl shadow-sky-500/10 backdrop-blur-sm">
          {/* Bright hairline along the top edge. */}
          <div
            aria-hidden="true"
            className="absolute inset-x-10 top-0 h-px bg-gradient-to-r from-transparent via-sky-300/70 to-transparent"
          />

          {/* ---------- Top bar ---------- */}
          <div className="flex items-center justify-between gap-4 border-b border-white/[0.07] px-4 py-2.5">
            <div className="flex min-w-0 items-center gap-3">
              <Sparkle className="size-4 shrink-0 text-sky-300" aria-hidden="true" />
              <div className="flex items-center gap-1.5 truncate text-[12px] text-slate-500">
                <span>Dashboards</span>
                <ChevronRight className="size-3" aria-hidden="true" />
                <span className="text-slate-300">Overview</span>
              </div>
            </div>

            <div className="hidden flex-1 justify-center sm:flex">
              <div className="flex w-full max-w-xs items-center gap-2 rounded-md border border-white/[0.07] bg-white/[0.03] px-2.5 py-1.5">
                <Search className="size-3.5 text-slate-500" aria-hidden="true" />
                <span className="text-[12px] text-slate-500">Search monitors…</span>
              </div>
            </div>

            <div className="flex shrink-0 items-center gap-1">
              {[Settings, Bell, Expand].map((Icon, i) => (
                <div
                  key={i}
                  className="flex size-7 items-center justify-center rounded-md text-slate-500"
                >
                  <Icon className="size-3.5" aria-hidden="true" />
                </div>
              ))}
            </div>
          </div>

          {/* ---------- Body ---------- */}
          <div className="flex">
            {/* Sidebar */}
            <aside className="hidden w-44 shrink-0 border-r border-white/[0.07] p-3 md:block">
              <p className="px-2 pb-1.5 text-[10px] font-medium uppercase tracking-wider text-slate-600">
                Favorites
              </p>
              <SidebarItem icon={Star} label="Overview" />
              <SidebarItem icon={Radio} label="Incidents" />

              <p className="px-2 pt-4 pb-1.5 text-[10px] font-medium uppercase tracking-wider text-slate-600">
                Workspace
              </p>
              <SidebarItem icon={LayoutDashboard} label="Monitors" active />
              <SidebarItem icon={FolderKanban} label="Projects" />
              <SidebarItem icon={Settings} label="Settings" />
            </aside>

            {/* Main table */}
            <div className="min-w-0 flex-1 p-4">
              <div className="mb-3 flex items-center justify-between">
                <h3 className="text-[13px] font-medium text-white">Monitor List</h3>
                <div className="flex items-center gap-1 text-[11px] text-slate-500">
                  Last 24h
                  <ChevronDown className="size-3" aria-hidden="true" />
                </div>
              </div>

              <div className="overflow-x-auto">
                <table className="w-full min-w-[420px] border-collapse">
                  <thead>
                    <tr className="border-b border-white/[0.07]">
                      {["Monitor", "Project", "Status", "Response", "Last check"].map(
                        (h) => (
                          <th
                            key={h}
                            scope="col"
                            className="px-2 py-2 text-left text-[11px] font-medium text-slate-500"
                          >
                            {h}
                          </th>
                        ),
                      )}
                    </tr>
                  </thead>
                  <tbody>
                    {MONITORS.map((m) => (
                      <tr
                        key={m.name}
                        className="border-b border-white/[0.04] last:border-0"
                      >
                        <td className="px-2 py-2.5 text-[12px] font-medium text-slate-200">
                          {m.name}
                        </td>
                        <td className="px-2 py-2.5 text-[12px] text-slate-500">
                          {m.project}
                        </td>
                        <td className="px-2 py-2.5">
                          <StatusPill status={m.status} />
                        </td>
                        <td className="px-2 py-2.5 font-mono text-[11px] tabular-nums text-slate-400">
                          {m.response}
                        </td>
                        <td className="px-2 py-2.5 font-mono text-[11px] tabular-nums text-slate-500">
                          {m.lastCheck}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>

            {/* Right panel */}
            <aside className="hidden w-56 shrink-0 border-l border-white/[0.07] p-4 lg:block">
              <p className="pb-3 text-[11px] font-medium text-slate-400">
                Notifications
              </p>
              <div className="space-y-3">
                {FEED.map((item) => (
                  <div key={item.initials} className="flex items-start gap-2.5">
                    <span
                      className={`flex size-6 shrink-0 items-center justify-center rounded-full text-[9px] font-semibold ${item.tint}`}
                    >
                      {item.initials}
                    </span>
                    <div className="min-w-0">
                      <p className="truncate text-[11px] leading-snug text-slate-300">
                        {item.text}
                      </p>
                      <p className="mt-0.5 font-mono text-[10px] text-slate-600">
                        {item.time} ago
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            </aside>
          </div>
        </div>

        {/* Fade the clipped edge into the canvas. */}
        <div
          aria-hidden="true"
          className="pointer-events-none absolute inset-x-0 bottom-0 h-24 bg-gradient-to-b from-transparent to-space-900"
        />
      </div>
    </section>
  );
}
