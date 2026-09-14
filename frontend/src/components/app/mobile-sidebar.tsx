"use client";

import { useState } from "react";
import { Menu, Sparkle } from "lucide-react";

import {
  Sheet,
  SheetContent,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { SidebarNav } from "./sidebar-nav";

/**
 * The navigation drawer for tablet and mobile.
 *
 * We hold the open/closed state ourselves rather than letting the Sheet manage
 * it, because we need to CLOSE the drawer when a link is tapped. Without that,
 * you navigate to the new page and the drawer stays open covering it — a small
 * detail that immediately makes an app feel broken.
 */
export function MobileSidebar() {
  const [open, setOpen] = useState(false);

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger
        render={
          <button
            type="button"
            aria-label="Open navigation menu"
            className="inline-flex size-9 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring lg:hidden"
          />
        }
      >
        <Menu className="size-4" />
      </SheetTrigger>

      <SheetContent side="left" className="w-64 bg-sidebar p-0">
        <div className="flex h-14 items-center gap-2 border-b border-border px-4">
          <Sparkle className="size-4" aria-hidden="true" />
          {/* Base UI dialogs need an accessible name; this one is visible. */}
          <SheetTitle className="text-sm font-semibold tracking-tight">
            API Monitor
          </SheetTitle>
        </div>

        <div className="p-3">
          {/* Closing on navigate is the whole reason this component owns state. */}
          <SidebarNav onNavigate={() => setOpen(false)} />
        </div>
      </SheetContent>
    </Sheet>
  );
}
