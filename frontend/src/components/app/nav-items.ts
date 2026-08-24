import {
  FolderKanban,
  LayoutDashboard,
  Radio,
  Settings,
  type LucideIcon,
} from "lucide-react";

/**
 * The dashboard navigation model.
 *
 * Plain data in its own module — no JSX, no "use client" — so that both the
 * Server Component sidebar and the Client Component mobile drawer can import
 * it without dragging each other across the server/client boundary.
 */

export type NavItem = {
  label: string;
  href: string;
  icon: LucideIcon;
};

export const NAV_ITEMS: readonly NavItem[] = [
  { label: "Dashboard", href: "/dashboard", icon: LayoutDashboard },
  { label: "Monitors", href: "/monitors", icon: Radio },
  { label: "Projects", href: "/projects", icon: FolderKanban },
  { label: "Settings", href: "/settings", icon: Settings },
] as const;
