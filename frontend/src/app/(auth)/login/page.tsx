import Link from "next/link";
import { ArrowRight, Construction } from "lucide-react";

/**
 * 🚧 PLACEHOLDER — the real sign-in form is built on Day 2 (PLAN.md task 45).
 *
 * This is deliberately honest about being unfinished rather than showing a
 * fake form that does nothing. A login box that silently ignores you is worse
 * than one that admits it is not wired up yet.
 *
 * The dashboard link below is a TEMPORARY development shortcut. It is removed
 * on Day 2 once real authentication protects that route.
 */
export default function LoginPage() {
  return (
    <div className="rounded-lg border border-border p-6">
      <div className="flex items-center gap-2 text-muted-foreground">
        <Construction className="size-4" aria-hidden="true" />
        <span className="text-xs font-medium uppercase tracking-wider">
          Not built yet
        </span>
      </div>

      <h1 className="mt-3 text-lg font-semibold tracking-tight">Sign in</h1>
      <p className="mt-1 text-sm text-muted-foreground">
        Email and password sign-in arrives on Day 2, together with the Go
        authentication API.
      </p>

      <div className="mt-6 space-y-2 border-t border-border pt-4">
        <p className="text-xs text-muted-foreground">
          Development shortcut — removed once sign-in is real:
        </p>
        <Link
          href="/dashboard"
          className="inline-flex items-center gap-1.5 text-sm font-medium hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
        >
          Preview the dashboard
          <ArrowRight className="size-3.5" aria-hidden="true" />
        </Link>
      </div>

      <p className="mt-6 text-sm text-muted-foreground">
        No account?{" "}
        <Link href="/register" className="text-foreground hover:underline">
          Create one
        </Link>
      </p>
    </div>
  );
}
