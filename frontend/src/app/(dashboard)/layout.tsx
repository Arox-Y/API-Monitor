import { AppSidebar } from "@/components/app/app-sidebar";
import { AppTopbar } from "@/components/app/app-topbar";

/**
 * The shell every dashboard page sits inside.
 *
 * In Next.js App Router, a layout.tsx wraps every page in its folder. The
 * pages render where {children} is. Navigating between /dashboard, /monitors
 * and /projects swaps only the children — the sidebar and topbar are NOT
 * re-rendered or re-mounted, which is why navigation feels instant.
 *
 * The (dashboard) folder name is in brackets, making it a "route group": it
 * groups files together without appearing in the URL. So this file lives at
 * app/(dashboard)/layout.tsx but the page is just /dashboard, not
 * /(dashboard)/dashboard.
 *
 * This is a Server Component, so none of this shell costs the browser any
 * JavaScript.
 */
export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex min-h-screen bg-background">
      <AppSidebar />

      {/* min-w-0 matters: without it a wide table inside would stretch this
          column and force the whole page to scroll sideways. */}
      <div className="flex min-w-0 flex-1 flex-col">
        <AppTopbar />
        <main className="flex-1 overflow-y-auto">{children}</main>
      </div>
    </div>
  );
}
