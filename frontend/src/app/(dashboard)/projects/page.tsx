import { PageHeader } from "@/components/app/page-header";
import {
  ProjectsContent,
  ProjectsHeaderAction,
} from "@/components/app/projects-content";

export default function ProjectsPage() {
  return (
    <div>
      <PageHeader
        title="Projects"
        description="Group your monitors and optionally publish a status page."
        action={<ProjectsHeaderAction />}
      />

      <div className="px-4 py-6 lg:px-6">
        <ProjectsContent />
      </div>
    </div>
  );
}
