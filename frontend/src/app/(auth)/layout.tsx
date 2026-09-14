import Link from "next/link";
import { Sparkle } from "lucide-react";

/**
 * Shared frame for the sign-in and sign-up pages.
 *
 * A centred single column: these pages have exactly one job, so there is no
 * sidebar and no navigation to distract from it.
 */
export default function AuthLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-background px-4 py-10">
      <Link
        href="/"
        className="mb-8 flex items-center gap-2 rounded-md focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-ring"
      >
        <Sparkle className="size-4" aria-hidden="true" />
        <span className="text-sm font-semibold tracking-tight">API Monitor</span>
      </Link>

      <div className="w-full max-w-sm">{children}</div>
    </div>
  );
}
