import Link from "next/link";
import { LogOut, User } from "lucide-react";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { MobileSidebar } from "./mobile-sidebar";

/**
 * The bar across the top of every dashboard page.
 *
 * Server Component. The mobile drawer and the dropdown menu are the only
 * interactive parts, and each is its own small client island.
 *
 * The user shown here is a placeholder until real login exists on Day 2.
 */
export function AppTopbar() {
  return (
    <header className="flex h-14 shrink-0 items-center justify-between gap-4 border-b border-border bg-background px-4 lg:px-6">
      <div className="flex items-center gap-2">
        <MobileSidebar />
      </div>

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
            <AvatarFallback className="text-xs">AR</AvatarFallback>
          </Avatar>
        </DropdownMenuTrigger>

        <DropdownMenuContent align="end" className="w-52">
          <div className="px-2 py-1.5">
            <p className="truncate text-sm font-medium">Ariyam</p>
            <p className="truncate text-xs text-muted-foreground">
              Not signed in yet
            </p>
          </div>

          <DropdownMenuSeparator />

          <DropdownMenuItem render={<Link href="/settings" />}>
            <User className="size-4" aria-hidden="true" />
            Settings
          </DropdownMenuItem>

          <DropdownMenuItem render={<Link href="/login" />}>
            <LogOut className="size-4" aria-hidden="true" />
            Sign in
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </header>
  );
}
