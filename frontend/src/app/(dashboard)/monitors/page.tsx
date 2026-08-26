import { PageHeader } from "@/components/app/page-header";
import {
  MonitorsContent,
  MonitorsHeaderAction,
} from "@/components/app/monitors-content";

export default function MonitorsPage() {
  return (
    <div>
      <PageHeader
        title="Monitors"
        description="Every endpoint being watched, across all your projects."
        action={<MonitorsHeaderAction />}
      />

      <div className="px-4 py-6 lg:px-6">
        <MonitorsContent />
      </div>
    </div>
  );
}
