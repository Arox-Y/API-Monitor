import { cn } from "@/lib/utils";

/**
 * One number on the dashboard.
 *
 * Deliberately plain: a border, a label, a number. No gradient, no shadow, no
 * coloured background. On a monitoring dashboard the NUMBER is the thing that
 * matters, and decoration around it only competes for attention.
 */
export function StatCard({
  label,
  value,
  hint,
  tone = "default",
}: {
  label: string;
  /** Already formatted for display — this component does not do formatting. */
  value: string;
  hint?: string;
  /** Colour the number only when the number itself carries meaning. */
  tone?: "default" | "up" | "down";
}) {
  return (
    <div className="rounded-lg border border-border p-4">
      <p className="text-sm text-muted-foreground">{label}</p>

      {/*
        tabular-nums makes every digit the same width. Without it, a value
        changing from 1,111 to 8,888 visibly shifts sideways, because in most
        fonts "1" is narrower than "8". On a dashboard that refreshes itself
        this looks like a twitch. It is a tiny detail that separates polished
        from sloppy.
      */}
      <p
        className={cn(
          "mt-2 text-2xl font-semibold tabular-nums",
          tone === "up" && "text-status-up",
          tone === "down" && "text-status-down",
        )}
      >
        {value}
      </p>

      {hint && <p className="mt-1 text-xs text-muted-foreground">{hint}</p>}
    </div>
  );
}
