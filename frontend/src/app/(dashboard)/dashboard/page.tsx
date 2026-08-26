import { DashboardContent } from "@/components/app/dashboard-content";
import { PageHeader } from "@/components/app/page-header";

/**
 * The dashboard page.
 *
 * A Server Component: it renders the header and hands the interactive part to
 * DashboardContent, which is the only piece that needs to run in the browser.
 *
 * The mock data used on Day 1 is gone - this now reads the real API.
 */
export default function DashboardPage() {
  return (
    <div>
      <PageHeader
        title="Dashboard"
        description="Health of every monitor across your projects."
      />

      <div className="px-4 py-6 lg:px-6">
        <DashboardContent />
      </div>
    </div>
  );
}
