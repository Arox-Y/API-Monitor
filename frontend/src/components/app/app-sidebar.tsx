import Link from "next/link";
import { Activity } from "lucide-react";

import { SidebarNav } from "./sidebar-nav";

/**
 * The fixed 240px sidebar shown on large screens.
 *
 * Server Component — only the link list inside it needs to be interactive.
 * On smaller screens this is hidden and MobileSidebar takes over.
 */
export function AppSidebar() {
  return (
    <aside className="hidden w-60 shrink-0 flex-col border-r border-border bg-sidebar lg:flex">
      {/* Brand. h-14 matches the topbar height so the two align exactly. */}
      <div className="flex h-14 items-center gap-2 border-b border-border px-4">
        <Activity className="size-4 text-foreground" aria-hidden="true" />
        <span className="text-sm font-semibold tracking-tight">API Monitor</span>
      </div>

      <div className="flex-1 overflow-y-auto p-3">
        <SidebarNav />
      </div>

      {/* Footer. Replaced with the real project count on Day 2. */}
      <div className="border-t border-border p-3">
        <Link
          href="/settings"
          className="block rounded-md px-2.5 py-2 text-xs text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
        >
          Free plan · 5 monitors
        </Link>
      </div>
    </aside>
  );
}
