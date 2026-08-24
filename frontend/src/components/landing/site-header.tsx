import Link from "next/link";
import { ChevronDown, Sparkle } from "lucide-react";

import { MobileNav } from "./mobile-nav";
import { NAV_LINKS } from "./nav-links";

/**
 * Sticky marketing header.
 *
 * Server Component — it renders links and static markup only. The mobile
 * drawer is imported as a separate Client Component so the interactivity
 * boundary stays as small as possible.
 */
export function SiteHeader() {
  return (
    <header className="sticky top-0 z-50 w-full border-b border-white/[0.06] bg-space-800/60 backdrop-blur-md">
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
        {/* Brand */}
        <Link
          href="/"
          className="group flex items-center gap-2 rounded-md focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-sky-400"
        >
          <Sparkle
            className="size-5 text-sky-300 transition-transform duration-300 group-hover:scale-110"
            style={{ filter: "drop-shadow(0 0 6px rgba(56,189,248,0.7))" }}
            aria-hidden="true"
          />
          <span className="text-[15px] font-semibold tracking-tight text-white">
            API Monitor
          </span>
        </Link>

        {/* Desktop navigation */}
        <nav className="hidden items-center gap-1 md:flex">
          {NAV_LINKS.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              className="inline-flex items-center gap-1 rounded-lg px-3 py-2 text-sm text-slate-400 transition-colors hover:text-white focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sky-400"
            >
              {link.label}
              {link.hasSubmenu && (
                <ChevronDown className="size-3.5 opacity-60" aria-hidden="true" />
              )}
            </Link>
          ))}
        </nav>

        {/* Actions */}
        <div className="flex items-center gap-2">
          <Link
            href="/login"
            className="hidden rounded-lg px-3 py-2 text-sm text-slate-300 transition-colors hover:text-white focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sky-400 sm:inline-flex"
          >
            Sign in
          </Link>

          <Link
            href="/register"
            className="hidden rounded-full bg-white px-4 py-2 text-sm font-medium text-slate-950 transition-all duration-200 hover:scale-[1.03] hover:bg-slate-100 hover:shadow-[0_0_20px_rgba(96,165,250,0.4)] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sky-400 sm:inline-flex"
          >
            Get Started
          </Link>

          <MobileNav />
        </div>
      </div>
    </header>
  );
}
