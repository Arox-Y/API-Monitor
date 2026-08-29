import { formatRelativeTime, formatStatusCode } from "@/lib/format";
import type { Check } from "@/types/api";

/**
 * The raw check history.
 *
 * The chart shows the shape of things; this table is what you read when you
 * need to know exactly what happened at 03:14.
 */
export function CheckHistory({ checks }: { checks: Check[] }) {
  if (checks.length === 0) {
    return (
      <div className="rounded-lg border border-dashed border-border px-6 py-8 text-center">
        <p className="text-sm text-muted-foreground">
          No checks yet. The first one runs within a minute of the monitor being
          enabled.
        </p>
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-lg border border-border">
      <div className="overflow-x-auto">
        <table className="w-full min-w-[420px] text-sm">
          <thead>
            <tr className="border-b border-border bg-muted/40">
              {/* scope="col" tells assistive technology these cells label their
                  whole column; without it a table is an unlabelled grid. */}
              <th scope="col" className="px-4 py-2.5 text-left font-medium text-muted-foreground">
                When
              </th>
              <th scope="col" className="px-4 py-2.5 text-left font-medium text-muted-foreground">
                Result
              </th>
              <th scope="col" className="px-4 py-2.5 text-left font-medium text-muted-foreground">
                Status
              </th>
              <th scope="col" className="px-4 py-2.5 text-right font-medium text-muted-foreground">
                Response
              </th>
            </tr>
          </thead>

          <tbody className="divide-y divide-border">
            {checks.map((check) => (
              <tr key={check.id}>
                <td className="px-4 py-2.5 font-mono text-xs tabular-nums text-muted-foreground">
                  {formatRelativeTime(check.checked_at)}
                </td>

                <td className="px-4 py-2.5">
                  {/* Text, not just colour. A green dot and a red dot look
                      identical to someone with red/green colour blindness. */}
                  <span
                    className={
                      check.success
                        ? "text-xs font-medium text-status-up"
                        : "text-xs font-medium text-status-down"
                    }
                  >
                    {check.success ? "Success" : "Failed"}
                  </span>
                  {check.error && (
                    <p className="mt-0.5 truncate font-mono text-xs text-muted-foreground">
                      {check.error}
                    </p>
                  )}
                </td>

                <td className="px-4 py-2.5 font-mono text-xs tabular-nums text-muted-foreground">
                  {/* null means nothing responded at all - a timeout or a DNS
                      failure. It renders as a dash, never as 0. */}
                  {formatStatusCode(check.status_code)}
                </td>

                <td className="px-4 py-2.5 text-right font-mono text-xs tabular-nums text-muted-foreground">
                  {check.success ? `${check.response_time_ms} ms` : "—"}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
