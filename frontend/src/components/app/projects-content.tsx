"use client";

import Link from "next/link";
import { ExternalLink, FolderKanban, Globe, Lock } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import {
  CreateProjectDialog,
  DeleteProjectDialog,
} from "@/components/app/project-dialogs";
import { EmptyState, ErrorState, TableSkeleton } from "@/components/app/states";
import { useProjects, useUpdateProject } from "@/hooks/use-projects";
import type { Project } from "@/types/api";

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
              <span className="tabular-nums">{project.monitor_count}</span>{" "}
              {project.monitor_count === 1 ? "monitor" : "monitors"}
            </p>

            {/* The status page link only appears when the project is public,
                because a private one would 404 and look broken. */}
            {project.is_public && (
              <Link
                href={`/status/${project.slug}`}
                target="_blank"
                rel="noreferrer"
                className="mt-1 inline-flex items-center gap-1 font-mono text-xs text-muted-foreground hover:text-foreground hover:underline"
              >
                /status/{project.slug}
                <ExternalLink className="size-3" aria-hidden="true" />
              </Link>
            )}
          </div>

          <div className="flex shrink-0 items-center gap-3">
            <PublicToggle project={project} />
            <DeleteProjectDialog project={project} />
          </div>
        </li>
      ))}
    </ul>
  );
}

/**
 * Publish or unpublish a project's status page.
 *
 * This is a PATCH with one field, not a dedicated endpoint. `is_public` is a
 * field, and PATCH is the verb for changing fields.
 */
function PublicToggle({ project }: { project: Project }) {
  const update = useUpdateProject(project.id);

  return (
    <div className="flex items-center gap-2">
      <Switch
        checked={project.is_public}
        disabled={update.isPending}
        // Without an accessible name a screen reader announces only "switch",
        // with no clue which project it belongs to.
        aria-label={`${project.is_public ? "Unpublish" : "Publish"} the status page for ${project.name}`}
        onCheckedChange={(checked) => update.mutate({ is_public: checked })}
      />
      <span className="hidden text-xs text-muted-foreground sm:inline">
        Public
      </span>
    </div>
  );
}

/** The header action, kept here so the page stays a Server Component. */
export function ProjectsHeaderAction() {
  return <CreateProjectDialog />;
}
