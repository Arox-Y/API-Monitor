"use client";

import { useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { Loader2, Plus, TriangleAlert } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { ApiError } from "@/lib/api";
import { projectSchema, type ProjectValues } from "@/lib/schemas";
import { useCreateProject, useDeleteProject } from "@/hooks/use-projects";
import type { Project } from "@/types/api";

/** "New project" button plus its dialog. */
export function CreateProjectDialog() {
  const [open, setOpen] = useState(false);
  const create = useCreateProject();

  const {
    register,
    handleSubmit,
    reset,
    setValue,
    watch,
    formState: { errors },
  } = useForm<ProjectValues>({
    resolver: zodResolver(projectSchema),
    defaultValues: { name: "", description: "", is_public: false },
  });

  const isPublic = watch("is_public");

  function onSubmit(values: ProjectValues) {
    create.mutate(values, {
      onSuccess: () => {
        // Reset the form as well as closing. Without this, reopening the
        // dialog would still show whatever was typed last time.
        reset();
        setOpen(false);
      },
    });
  }

  const generalError =
    create.error instanceof ApiError ? create.error.message : null;

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button />}>
        <Plus className="size-4" aria-hidden="true" />
        New project
      </DialogTrigger>

      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>New project</DialogTitle>
          <DialogDescription>
            A project groups monitors together and can have its own public
            status page.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" noValidate>
          {generalError && (
            <p role="alert" className="text-sm text-status-down">
              {generalError}
            </p>
          )}

          <div className="space-y-1.5">
            <Label htmlFor="project-name">Name</Label>
            <Input
              id="project-name"
              placeholder="Acme API"
              aria-invalid={Boolean(errors.name)}
              {...register("name")}
            />
            {errors.name && (
              <p className="text-xs text-status-down">{errors.name.message}</p>
            )}
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="project-description">Description</Label>
            <Input
              id="project-description"
              placeholder="Production services"
              aria-invalid={Boolean(errors.description)}
              {...register("description")}
            />
            {errors.description && (
              <p className="text-xs text-status-down">
                {errors.description.message}
              </p>
            )}
          </div>

          <div className="flex items-center justify-between rounded-md border border-border px-3 py-2.5">
            <div>
              <Label htmlFor="project-public" className="cursor-pointer">
                Public status page
              </Label>
              <p className="mt-0.5 text-xs text-muted-foreground">
                Anyone with the link can see this project&apos;s status.
              </p>
            </div>
            {/* Switch is not a native input, so react-hook-form cannot register
                it directly. We drive it with watch/setValue instead. */}
            <Switch
              id="project-public"
              checked={isPublic ?? false}
              onCheckedChange={(checked) => setValue("is_public", checked)}
            />
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => setOpen(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={create.isPending}>
              {create.isPending && (
                <Loader2 className="size-4 animate-spin" aria-hidden="true" />
              )}
              Create project
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

/**
 * Delete confirmation.
 *
 * Deleting a project cascades to its monitors, checks and incidents. An action
 * that destroys that much data must never be one unguarded click, and the
 * dialog says exactly what will be lost rather than a vague "Are you sure?".
 */
export function DeleteProjectDialog({ project }: { project: Project }) {
  const [open, setOpen] = useState(false);
  const remove = useDeleteProject();

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger
        render={
          <Button variant="ghost" size="sm" aria-label={`Delete ${project.name}`} />
        }
      >
        Delete
      </DialogTrigger>

      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <TriangleAlert className="size-4 text-status-down" aria-hidden="true" />
            Delete {project.name}?
          </DialogTitle>
          <DialogDescription>
            This permanently deletes the project and its{" "}
            <strong className="text-foreground">
              {project.monitor_count}{" "}
              {project.monitor_count === 1 ? "monitor" : "monitors"}
            </strong>
            , along with all recorded checks and incidents. This cannot be
            undone.
          </DialogDescription>
        </DialogHeader>

        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Cancel
          </Button>
          <Button
            variant="destructive"
            disabled={remove.isPending}
            onClick={() =>
              remove.mutate(project.id, { onSuccess: () => setOpen(false) })
            }
          >
            {remove.isPending && (
              <Loader2 className="size-4 animate-spin" aria-hidden="true" />
            )}
            Delete project
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
