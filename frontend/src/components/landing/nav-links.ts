/**
 * Shared navigation model for the marketing header.
 *
 * Kept in its own module (with no "use client" and no JSX) so that both the
 * Server Component header and the Client Component mobile drawer can import it
 * without dragging one into the other's rendering environment.
 */

export type NavLink = {
  label: string;
  href: string;
  /** Renders a chevron to signal a submenu. The menu itself is post-MVP. */
  hasSubmenu?: boolean;
};

export const NAV_LINKS: readonly NavLink[] = [
  { label: "Features", href: "#features" },
  { label: "How it works", href: "#how-it-works" },
  { label: "Docs", href: "#docs", hasSubmenu: true },
  { label: "Pricing", href: "#pricing", hasSubmenu: true },
] as const;
