"use client";

import { FolderKanban, Globe, Lock } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  CreateProjectDialog,
  DeleteProjectDialog,
} from "@/components/app/project-dialogs";
import { EmptyState, ErrorState, TableSkeleton } from "@/components/app/states";
import { useProjects } from "@/hooks/use-projects";

/**
 * The projects list, reading live data.
 *
 * Handles all four states: loading, error, empty, success.
 */
export function ProjectsContent() {
  const { data: projects, isPending, isError, error, refetch } = useProjects();

  if (isPending) {
    return <TableSkeleton rows={3} />;
  }

  if (isError) {
    return (
      <ErrorState
        title="Could not load your projects"
        description={error.message}
        onRetry={
          <Button variant="outline" onClick={() => refetch()}>
            Try again
          </Button>
        }
      />
    );
  }

  if (projects.length === 0) {
    return (
      <EmptyState
        icon={FolderKanban}
        title="No projects yet"
        description="A project groups your monitors together. Create one to get started."
        action={<CreateProjectDialog />}
      />
    );
  }

  return (
    <ul className="divide-y divide-border rounded-lg border border-border">
      {projects.map((project) => (
        <li
          key={project.id}
          className="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between"
        >
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <span className="font-medium">{project.name}</span>

              {/* Public vs private uses an ICON AND TEXT, never colour alone. */}
              {project.is_public ? (
                <span className="inline-flex items-center gap-1 rounded-full border border-border px-2 py-0.5 text-xs text-muted-foreground">
                  <Globe className="size-3" aria-hidden="true" />
                  Public
                </span>
              ) : (
                <span className="inline-flex items-center gap-1 rounded-full border border-border px-2 py-0.5 text-xs text-muted-foreground">
                  <Lock className="size-3" aria-hidden="true" />
                  Private
                </span>
              )}
            </div>

            {project.description && (
              <p className="mt-1 truncate text-sm text-muted-foreground">
                {project.description}
              </p>
            )}

            <p className="mt-1 font-mono text-xs text-muted-foreground">
              /status/{project.slug} ·{" "}
              <span className="tabular-nums">{project.monitor_count}</span>{" "}
              {project.monitor_count === 1 ? "monitor" : "monitors"}
            </p>
          </div>

          <div className="flex shrink-0 items-center gap-1">
            <DeleteProjectDialog project={project} />
          </div>
        </li>
      ))}
    </ul>
  );
}

/** The header action, kept here so the page stays a Server Component. */
export function ProjectsHeaderAction() {
  return <CreateProjectDialog />;
}
