import { CircleAlert, CircleCheck, CircleDashed, CircleX } from "lucide-react";

import { cn } from "@/lib/utils";
import type { MonitorStatus, OverallStatus } from "@/types/api";

/**
 * The single place monitor status is turned into something visible.
 *
 * Every status shows an ICON, a COLOUR and a TEXT LABEL — never colour alone.
 *
 * Why: roughly 1 in 12 men has some form of red/green colour blindness. A red
 * dot and a green dot look identical to them. Colour is a nice extra hint, but
 * it can never be the only way to tell two states apart. This is WCAG rule
 * 1.4.1 ("Use of Color").
 *
 * Because every screen uses THIS component, the visual language cannot drift
 * apart between the dashboard, the monitor list and the status page.
 */

type Status = MonitorStatus | OverallStatus;

type StatusConfig = {
  Icon: React.ComponentType<{ className?: string }>;
  label: string;
  /** Badge styling: border, faint background, text colour. */
  badgeClassName: string;
  /** Solid fill for the small dot variant. */
  dotClassName: string;
};

const STATUS_CONFIG = {
  up: {
    Icon: CircleCheck,
    label: "Up",
    badgeClassName: "border-status-up/25 bg-status-up/10 text-status-up",
    dotClassName: "bg-status-up",
  },
  down: {
    Icon: CircleX,
    label: "Down",
    badgeClassName: "border-status-down/25 bg-status-down/10 text-status-down",
    dotClassName: "bg-status-down",
  },
  pending: {
    Icon: CircleDashed,
    label: "Pending",
    badgeClassName:
      "border-status-pending/25 bg-status-pending/10 text-status-pending",
    dotClassName: "bg-status-pending",
  },
  degraded: {
    Icon: CircleAlert,
    label: "Degraded",
    badgeClassName:
      "border-status-degraded/25 bg-status-degraded/10 text-status-degraded",
    dotClassName: "bg-status-degraded",
  },
} as const satisfies Record<Status, StatusConfig>;

export function StatusBadge({
  status,
  className,
}: {
  status: Status;
  className?: string;
}) {
  const { Icon, label, badgeClassName } = STATUS_CONFIG[status];

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium",
        badgeClassName,
        className,
      )}
    >
      {/* aria-hidden because the text label right next to it already says
          the same thing — a screen reader would otherwise announce it twice. */}
      <Icon className="size-3" aria-hidden="true" />
      {label}
    </span>
  );
}

/** A smaller dot + label, for tight spaces like table rows. */
export function StatusDot({ status }: { status: Status }) {
  const { label, dotClassName } = STATUS_CONFIG[status];

  return (
    <span className="inline-flex items-center gap-2 text-sm">
      <span className={cn("size-2 rounded-full", dotClassName)} aria-hidden="true" />
      {label}
    </span>
  );
}
