import { PageHeader } from "@/components/app/page-header";
import {
  BackToMonitors,
  MonitorDetailContent,
} from "@/components/app/monitor-detail-content";

/**
 * One monitor's detail page, at /monitors/{id}.
 *
 * [id] in the folder name makes this a dynamic route: any value in that
 * position matches, and Next.js hands it over in `params`.
 *
 * In Next.js 15 `params` is a PROMISE and must be awaited. That changed in
 * this major version, so older examples showing a plain object are out of date.
 */
export default async function MonitorDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;

  return (
    <div>
      <PageHeader
        title="Monitor"
        description="Response times, incidents and raw check history."
        action={<BackToMonitors />}
      />

      <div className="px-4 py-6 lg:px-6">
        <MonitorDetailContent monitorId={id} />
      </div>
    </div>
  );
}
