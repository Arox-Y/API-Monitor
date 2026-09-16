import { notFound } from "next/navigation";
import { CircleAlert, CircleCheck, CircleX, Sparkle } from "lucide-react";

/**
 * The PUBLIC status page, at /status/{slug}.
 *
 * This is a SERVER COMPONENT that fetches on the server, unlike the dashboard
 * which fetches in the browser with TanStack Query. The difference is
 * deliberate:
 *
 *   - the dashboard is private, interactive, and refreshes itself while you
 *     watch it, so client-side fetching earns its keep
 *   - a status page is public and read-only. Rendering it on the server means
 *     it arrives complete, with no loading flash, works with JavaScript
 *     disabled, and can be read by search engines and link previews
 *
 * Nothing here requires a session. The route is also absent from
 * middleware.ts's matcher, so no redirect ever applies to it.
 */

type PublicService = {
  name: string;
  status: "up" | "down" | "pending";
  uptime_percentage_24h: number | null;
};

type PublicIncident = {
  service_name: string;
  started_at: string;
};

type PublicStatus = {
  project_name: string;
  overall_status: "up" | "degraded" | "down";
  uptime_percentage_24h: number | null;
  services: PublicService[];
  active_incidents: PublicIncident[];
};

// This is a SERVER component, so this fetch runs inside the Next.js
// container itself - NOT in the visitor's browser. NEXT_PUBLIC_API_URL is
// baked in at build time for the BROWSER's benefit (e.g. "http://localhost:8080",
// reachable via the host port mapping); from inside the frontend container
// that same address is the frontend container's own loopback, where nothing
// is listening. INTERNAL_API_URL is a plain runtime env var (read fresh on
// every request, no rebuild needed) so docker-compose.yml can point the
// server at the backend's container-network address ("http://backend:8080")
// while the browser keeps using the host-mapped one.
const API_BASE =
  process.env.INTERNAL_API_URL ??
  process.env.NEXT_PUBLIC_API_URL ??
  "http://localhost:8080";

async function fetchStatus(slug: string): Promise<PublicStatus | null> {
  try {
    const response = await fetch(
      `${API_BASE}/api/v1/public/status/${encodeURIComponent(slug)}`,
      {
        // Re-fetch at most every 30 seconds. A status page linked from
        // somewhere busy would otherwise hit the database on every view.
        next: { revalidate: 30 },
      },
    );

    if (!response.ok) return null;
    return (await response.json()) as PublicStatus;
  } catch {
    // The backend being unreachable is not the same as the page not existing,
    // but from a visitor's point of view there is nothing useful to say either
    // way, so both end up as a 404.
    return null;
  }
}

const OVERALL = {
  up: {
    Icon: CircleCheck,
    title: "All systems operational",
    className: "text-status-up",
  },
  degraded: {
    Icon: CircleAlert,
    title: "Partial outage",
    className: "text-status-degraded",
  },
  down: {
    Icon: CircleX,
    title: "Major outage",
    className: "text-status-down",
  },
} as const;

const SERVICE = {
  up: { label: "Operational", className: "text-status-up" },
  down: { label: "Down", className: "text-status-down" },
  pending: { label: "Not yet checked", className: "text-muted-foreground" },
} as const;

function formatUptime(value: number | null): string {
  // null means nothing has been measured. It must never render as 0%.
  return value === null ? "—" : `${value.toFixed(2)}%`;
}

export default async function StatusPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const status = await fetchStatus(slug);

  // A private project and a nonexistent one both arrive here as null, so the
  // page cannot be used to discover which projects exist.
  if (!status) notFound();

  const overall = OVERALL[status.overall_status];

  return (
    <div className="min-h-screen bg-background">
      <div className="mx-auto max-w-2xl px-4 py-12 sm:py-16">
        <header className="flex items-center gap-2 text-sm text-muted-foreground">
          <Sparkle className="size-4" aria-hidden="true" />
          API Monitor
        </header>

        <h1 className="mt-8 text-xl font-semibold tracking-tight">
          {status.project_name}
        </h1>

        {/* The headline. Icon AND text, never colour alone. */}
        <div className="mt-6 flex items-center gap-2.5 rounded-lg border border-border p-4">
          <overall.Icon
            className={`size-5 shrink-0 ${overall.className}`}
            aria-hidden="true"
          />
          <div>
            <p className="font-medium">{overall.title}</p>
            <p className="mt-0.5 text-sm text-muted-foreground">
              {formatUptime(status.uptime_percentage_24h)} uptime over the last
              24 hours
            </p>
          </div>
        </div>

        {status.active_incidents.length > 0 && (
          <section className="mt-8" aria-labelledby="incidents-heading">
            <h2
              id="incidents-heading"
              className="text-sm font-medium text-status-down"
            >
              Active incidents
            </h2>
            <ul className="mt-3 divide-y divide-border rounded-lg border border-status-down/25">
              {status.active_incidents.map((incident) => (
                <li
                  key={`${incident.service_name}-${incident.started_at}`}
                  className="p-3"
                >
                  <p className="text-sm font-medium">{incident.service_name}</p>
                  <p className="mt-0.5 font-mono text-xs text-muted-foreground">
                    since{" "}
                    {new Date(incident.started_at).toLocaleString([], {
                      dateStyle: "medium",
                      timeStyle: "short",
                    })}
                  </p>
                </li>
              ))}
            </ul>
          </section>
        )}

        <section className="mt-8" aria-labelledby="services-heading">
          <h2 id="services-heading" className="text-sm font-medium">
            Services
          </h2>

          {status.services.length === 0 ? (
            <p className="mt-3 rounded-lg border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
              No services are being monitored yet.
            </p>
          ) : (
            <ul className="mt-3 divide-y divide-border rounded-lg border border-border">
              {status.services.map((service) => {
                const config = SERVICE[service.status];
                return (
                  <li
                    key={service.name}
                    className="flex items-center justify-between gap-4 p-3"
                  >
                    {/* Only the NAME. The URL is deliberately not in the API
                        response at all, so it cannot be rendered here. */}
                    <span className="truncate text-sm">{service.name}</span>

                    <div className="flex shrink-0 items-center gap-4">
                      <span className="font-mono text-xs tabular-nums text-muted-foreground">
                        {formatUptime(service.uptime_percentage_24h)}
                      </span>
                      <span className={`text-xs font-medium ${config.className}`}>
                        {config.label}
                      </span>
                    </div>
                  </li>
                );
              })}
            </ul>
          )}
        </section>

        <footer className="mt-12 border-t border-border pt-6 text-xs text-muted-foreground">
          Status updates automatically. Powered by API Monitor.
        </footer>
      </div>
    </div>
  );
}

/**
 * Page metadata for the browser tab and link previews.
 *
 * The project name is safe to show - it is already visible on the page.
 * Nothing internal goes in here.
 */
export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const status = await fetchStatus(slug);

  if (!status) return { title: "Status page not found" };

  return {
    title: `${status.project_name} Status`,
    description: `Current operational status for ${status.project_name}.`,
  };
}
