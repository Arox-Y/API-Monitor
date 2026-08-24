import { TriangleAlert, type LucideIcon } from "lucide-react";

import { cn } from "@/lib/utils";

/**
 * The shared loading / empty / error states.
 *
 * Every part of the UI that loads data must handle FOUR situations:
 *   loading · success · empty · error
 *
 * A component that only draws the success case is not finished. In real use
 * the other three happen constantly — slow connections, new accounts, servers
 * having a bad day.
 */

/* -------------------------------------------------------------------------
   LOADING — skeletons, not spinners
   ------------------------------------------------------------------------- */

/**
 * A grey block roughly the size of the content that is coming.
 *
 * Why not a spinner? A spinner says "something is happening" but nothing about
 * what. A skeleton keeps the page the right shape, so when the real content
 * arrives nothing jumps around.
 *
 * The catch: a skeleton with the WRONG size is worse than none, because it
 * causes the very jump it was meant to prevent. Match the real dimensions.
 */
export function Skeleton({ className }: { className?: string }) {
  return (
    <div
      className={cn("animate-pulse rounded-md bg-muted", className)}
      aria-hidden="true"
    />
  );
}

/** Loading placeholder shaped like the stat cards. */
export function StatCardSkeleton() {
  return (
    <div className="rounded-lg border border-border p-4">
      <Skeleton className="h-3.5 w-24" />
      <Skeleton className="mt-3 h-7 w-16" />
    </div>
  );
}

/** Loading placeholder shaped like a table. */
export function TableSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <div className="divide-y divide-border rounded-lg border border-border">
      {Array.from({ length: rows }).map((_, index) => (
        <div key={index} className="flex items-center gap-4 p-3">
          <Skeleton className="h-4 w-40" />
          <Skeleton className="h-4 w-20" />
          <Skeleton className="ml-auto h-4 w-16" />
        </div>
      ))}
    </div>
  );
}

/* -------------------------------------------------------------------------
   EMPTY
   ------------------------------------------------------------------------- */

/**
 * Shown when there is genuinely nothing yet.
 *
 * Three things every empty state needs:
 *   1. what this section is
 *   2. why it is empty
 *   3. ONE clear next action
 *
 * A blank area with no explanation reads as a bug.
 *
 * Important: "nothing yet" and "your filter matched nothing" are different
 * situations and need different wording. Telling someone with 40 monitors
 * "No monitors yet" because their search found none is simply wrong.
 */
export function EmptyState({
  icon: Icon,
  title,
  description,
  action,
}: {
  icon?: LucideIcon;
  title: string;
  description: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border px-6 py-12 text-center">
      {Icon && (
        <Icon
          className="mb-3 size-5 text-muted-foreground"
          aria-hidden="true"
        />
      )}
      <p className="text-sm font-medium">{title}</p>
      <p className="mt-1 max-w-sm text-sm text-muted-foreground">
        {description}
      </p>
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
}

/* -------------------------------------------------------------------------
   ERROR
   ------------------------------------------------------------------------- */

/**
 * Shown when loading failed.
 *
 * Rules:
 *   - say what failed, in plain language
 *   - always offer a way to try again
 *   - never show a raw error or stack trace: it means nothing to the user and
 *     can leak internal details like server names
 *   - never just "Something went wrong"
 */
export function ErrorState({
  title = "Could not load this data",
  description = "Something went wrong while talking to the server.",
  onRetry,
}: {
  title?: string;
  description?: string;
  onRetry?: React.ReactNode;
}) {
  return (
    <div
      // role="alert" makes a screen reader announce this immediately, rather
      // than the user sitting in silence wondering why nothing loaded.
      role="alert"
      className="flex flex-col items-center justify-center rounded-lg border border-status-down/25 bg-status-down/5 px-6 py-10 text-center"
    >
      <TriangleAlert
        className="mb-3 size-5 text-status-down"
        aria-hidden="true"
      />
      <p className="text-sm font-medium">{title}</p>
      <p className="mt-1 max-w-sm text-sm text-muted-foreground">
        {description}
      </p>
      {onRetry && <div className="mt-4">{onRetry}</div>}
    </div>
  );
}
