"use client";

import Link from "next/link";
import { Menu } from "lucide-react";

import {
  Sheet,
  SheetContent,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { NAV_LINKS } from "./nav-links";

/**
 * The only Client Component on the landing page.
 *
 * It is deliberately small: the header around it stays a Server Component, so
 * the interactive JavaScript shipped for the whole page is just this drawer.
 * This is the "push the client boundary down, not up" rule from
 * docs/FRONTEND_DESIGN.md §7.
 */
export function MobileNav() {
  return (
    <Sheet>
      <SheetTrigger
        render={
          <button
            type="button"
            aria-label="Open navigation menu"
            className="inline-flex size-9 items-center justify-center rounded-lg border border-white/10 text-slate-300 transition-colors hover:bg-white/10 hover:text-white focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sky-400 md:hidden"
          />
        }
      >
        <Menu className="size-4" />
      </SheetTrigger>

      <SheetContent
        side="right"
        className="border-white/10 bg-space-800 text-slate-200"
      >
        {/* Base UI dialogs need an accessible name. Visually hidden is fine. */}
        <SheetTitle className="sr-only">Navigation</SheetTitle>

        <nav className="mt-10 flex flex-col gap-1 px-4">
          {NAV_LINKS.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              className="rounded-lg px-3 py-2.5 text-sm text-slate-300 transition-colors hover:bg-white/5 hover:text-white"
            >
              {link.label}
            </Link>
          ))}

          <div className="my-3 h-px bg-white/10" />

          <Link
            href="/login"
            className="rounded-lg px-3 py-2.5 text-sm text-slate-300 transition-colors hover:bg-white/5 hover:text-white"
          >
            Sign in
          </Link>
          <Link
            href="/register"
            className="mt-1 rounded-full bg-white px-4 py-2.5 text-center text-sm font-medium text-slate-950 transition-colors hover:bg-slate-200"
          >
            Get Started
          </Link>
        </nav>
      </SheetContent>
    </Sheet>
  );
}
