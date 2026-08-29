"use client";

import {
  CartesianGrid,
  Line,
  LineChart,
  ReferenceDot,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

import type { Check } from "@/types/api";

/**
 * Response time over the last 24 hours.
 *
 * Design decisions, all deliberate:
 *
 * - ONE series colour, not five. A single line showing one measurement needs
 *   no palette; multiple colours would imply categories that do not exist.
 * - Failed checks are marked as red dots ON the line rather than as a second
 *   series, because they are the same measurement in a different state.
 * - No gradient fill, no drop shadow, no animation on load. This is a chart to
 *   read at 3am during an outage, not a slide.
 */

type ChartPoint = {
  time: number;
  label: string;
  responseTime: number;
  success: boolean;
  statusCode: number | null;
};

export function ResponseTimeChart({ checks }: { checks: Check[] }) {
  if (checks.length === 0) {
    return (
      <div className="flex h-56 items-center justify-center rounded-lg border border-dashed border-border">
        <p className="text-sm text-muted-foreground">
          No checks recorded yet. Data appears once the monitor runs.
        </p>
      </div>
    );
  }

  // The API returns newest first, which is right for a table and wrong for a
  // chart: time must run left to right. Reversing a copy avoids mutating the
  // array the cache is holding.
  const data: ChartPoint[] = [...checks].reverse().map((check) => {
    const date = new Date(check.checked_at);
    return {
      time: date.getTime(),
      label: date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }),
      // A failed check has no meaningful duration, so it is plotted at 0
      // rather than at whatever the connection spent before giving up.
      responseTime: check.success ? check.response_time_ms : 0,
      success: check.success,
      statusCode: check.status_code,
    };
  });

  const failures = data.filter((point) => !point.success);

  return (
    <div className="rounded-lg border border-border p-4">
      <div className="mb-4 flex items-baseline justify-between">
        <h3 className="text-sm font-medium">Response time</h3>
        <span className="font-mono text-xs tabular-nums text-muted-foreground">
          {checks.length} checks · last 24h
        </span>
      </div>

      {/* ResponsiveContainer makes the chart fill its parent, so it adapts to
          the screen instead of having a hardcoded pixel width. */}
      <ResponsiveContainer width="100%" height={220}>
        <LineChart data={data} margin={{ top: 4, right: 8, bottom: 4, left: -8 }}>
          {/* Horizontal lines only. Vertical grid lines add clutter without
              helping anyone read a value off the axis. */}
          <CartesianGrid
            strokeDasharray="3 3"
            vertical={false}
            stroke="var(--border)"
          />

          <XAxis
            dataKey="label"
            tick={{ fontSize: 11, fill: "var(--muted-foreground)" }}
            tickLine={false}
            axisLine={false}
            minTickGap={40}
          />

          <YAxis
            tick={{ fontSize: 11, fill: "var(--muted-foreground)" }}
            tickLine={false}
            axisLine={false}
            width={48}
            unit="ms"
          />

          <Tooltip
            cursor={{ stroke: "var(--border)" }}
            content={<ChartTooltip />}
          />

          <Line
            type="monotone"
            dataKey="responseTime"
            stroke="var(--status-up)"
            strokeWidth={2}
            // Hide dots on every point - with 100 checks they merge into a
            // solid band and hide the line itself.
            dot={false}
            activeDot={{ r: 4 }}
            isAnimationActive={false}
          />

          {/* Failures marked individually, so an outage is visible at a glance
              rather than looking like a dip. */}
          {failures.map((point) => (
            <ReferenceDot
              key={point.time}
              x={point.label}
              y={0}
              r={4}
              fill="var(--status-down)"
              stroke="var(--background)"
              strokeWidth={1.5}
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}

/** Custom tooltip: shows what actually happened, not just a number. */
function ChartTooltip({
  active,
  payload,
}: {
  active?: boolean;
  payload?: Array<{ payload: ChartPoint }>;
}) {
  if (!active || !payload?.length) return null;

  const point = payload[0].payload;

  return (
    <div className="rounded-md border border-border bg-popover px-2.5 py-2 text-xs shadow-sm">
      <p className="font-mono tabular-nums">{point.label}</p>
      {point.success ? (
        <p className="mt-1 font-mono tabular-nums text-status-up">
          {point.responseTime} ms · {point.statusCode}
        </p>
      ) : (
        <p className="mt-1 font-mono text-status-down">
          {/* status_code is null when nothing responded at all. Saying
              "no response" is honest; showing 0 would be a lie. */}
          failed · {point.statusCode ?? "no response"}
        </p>
      )}
    </div>
  );
}
