"use client";

import Link from "next/link";
import { LogOut, Moon, Sun, User } from "lucide-react";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useTheme } from "@/components/theme-provider";
import { useLogout, useMe } from "@/hooks/use-auth";
import { MobileSidebar } from "./mobile-sidebar";

/**
 * The bar across the top of every dashboard page.
 *
 * A Client Component - it reads the signed-in user and the theme, both of
 * which only exist in the browser. Everything static around it (the layout,
 * the page shells) stays server-rendered; this is the one piece of chrome
 * that genuinely needs to run on the client.
 *
 * Previously showed a hardcoded "Ariyam" / "Not signed in yet" placeholder
 * with a "Sign in" link that did not sign anyone out - a leftover from
 * before real authentication existed, never connected up afterwards.
 */
export function AppTopbar() {
  const { data: user } = useMe();
  const logout = useLogout();
  const { resolvedTheme, setTheme } = useTheme();

  // Two initials from the account name, e.g. "Ada Lovelace" -> "AL". Falls
  // back to a generic mark while the user is still loading.
  const initials =
    user?.name
      .split(" ")
      .map((part) => part[0])
      .join("")
      .slice(0, 2)
      .toUpperCase() || "?";

  return (
    <header className="flex h-14 shrink-0 items-center justify-between gap-4 border-b border-border bg-background px-4 lg:px-6">
      <div className="flex items-center gap-2">
        <MobileSidebar />
      </div>

      <div className="flex items-center gap-1">
        <Button
          type="button"
          variant="ghost"
          size="icon"
          aria-label={
            resolvedTheme === "dark" ? "Switch to light theme" : "Switch to dark theme"
          }
          onClick={() => setTheme(resolvedTheme === "dark" ? "light" : "dark")}
        >
          {/* Both icon AND aria-label communicate the action - never an icon
              alone, the same rule the status badges follow. */}
          {resolvedTheme === "dark" ? (
            <Sun className="size-4" aria-hidden="true" />
          ) : (
            <Moon className="size-4" aria-hidden="true" />
          )}
        </Button>

        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <button
                type="button"
                aria-label="Account menu"
                className="flex items-center gap-2 rounded-md p-1 transition-colors hover:bg-accent focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
              />
            }
          >
            <Avatar className="size-7">
              <AvatarFallback className="text-xs">{initials}</AvatarFallback>
            </Avatar>
          </DropdownMenuTrigger>

          <DropdownMenuContent align="end" className="w-52">
            <div className="px-2 py-1.5">
              <p className="truncate text-sm font-medium">
                {user?.name ?? "Loading..."}
              </p>
              <p className="truncate text-xs text-muted-foreground">
                {user?.email ?? ""}
              </p>
            </div>

            <DropdownMenuSeparator />

            <DropdownMenuItem render={<Link href="/settings" />}>
              <User className="size-4" aria-hidden="true" />
              Settings
            </DropdownMenuItem>

            <DropdownMenuItem
              onClick={() => logout.mutate()}
              disabled={logout.isPending}
            >
              <LogOut className="size-4" aria-hidden="true" />
              {logout.isPending ? "Signing out..." : "Sign out"}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  );
}
