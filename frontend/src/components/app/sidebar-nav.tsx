"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

import { cn } from "@/lib/utils";
import { NAV_ITEMS } from "./nav-items";

/**
 * The list of navigation links, with the current page highlighted.
 *
 * This is a Client Component for exactly one reason: usePathname() needs to
 * know which URL the browser is on, and that is browser state. Everything
 * around it — the sidebar frame, the logo, the layout — stays server-rendered.
 *
 * Reused by both the desktop sidebar and the mobile drawer, so "which link is
 * active" is decided in one place only.
 */
export function SidebarNav({ onNavigate }: { onNavigate?: () => void }) {
  const pathname = usePathname();

  return (
    <nav className="flex flex-col gap-0.5" aria-label="Main navigation">
      {NAV_ITEMS.map((item) => {
        // startsWith, not ===, so /monitors/abc123 still highlights "Monitors".
        const isActive =
          pathname === item.href || pathname.startsWith(`${item.href}/`);

        return (
          <Link
            key={item.href}
            href={item.href}
            onClick={onNavigate}
            // aria-current tells a screen reader which page you are on.
            // Without it, the highlight is visible but silent.
            aria-current={isActive ? "page" : undefined}
            className={cn(
              "flex items-center gap-2.5 rounded-md px-2.5 py-2 text-sm transition-colors",
              "focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring",
              isActive
                ? "bg-sidebar-accent font-medium text-sidebar-accent-foreground"
                : "text-muted-foreground hover:bg-sidebar-accent/60 hover:text-foreground",
            )}
          >
            <item.icon className="size-4 shrink-0" aria-hidden="true" />
            {item.label}
          </Link>
        );
      })}
    </nav>
  );
}
