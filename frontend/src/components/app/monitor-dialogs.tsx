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
import { ApiError } from "@/lib/api";
import { monitorSchema, type MonitorValues } from "@/lib/schemas";
import { useCreateMonitor, useDeleteMonitor } from "@/hooks/use-monitors";
import { useProjects } from "@/hooks/use-projects";
import type { Monitor } from "@/types/api";

/**
 * "New monitor" button and dialog.
 *
 * A monitor must belong to a project, so the dialog is disabled until at
 * least one exists - and says why, rather than failing after submission.
 */
export function CreateMonitorDialog() {
  const [open, setOpen] = useState(false);
  const [projectId, setProjectId] = useState("");
  const { data: projects } = useProjects();

  const effectiveProjectId = projectId || projects?.[0]?.id || "";
  const create = useCreateMonitor(effectiveProjectId);

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<MonitorValues>({
    resolver: zodResolver(monitorSchema),
    defaultValues: {
      name: "",
      url: "",
      method: "GET",
      interval_seconds: 60,
      timeout_seconds: 10,
      expected_status: 200,
      enabled: true,
    },
  });

  const hasProjects = (projects?.length ?? 0) > 0;

  function onSubmit(values: MonitorValues) {
    create.mutate(values, {
      onSuccess: () => {
        reset();
        setOpen(false);
      },
    });
  }

  const generalError =
    create.error instanceof ApiError && create.error.details.length === 0
      ? create.error.message
      : null;

  if (!hasProjects) {
    return (
      <Button disabled title="Create a project first">
        <Plus className="size-4" aria-hidden="true" />
        New monitor
      </Button>
    );
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button />}>
        <Plus className="size-4" aria-hidden="true" />
        New monitor
      </DialogTrigger>

      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>New monitor</DialogTitle>
          <DialogDescription>
            We will check this URL on a schedule and record the result.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" noValidate>
          {generalError && (
            <p role="alert" className="text-sm text-status-down">
              {generalError}
            </p>
          )}

          <div className="space-y-1.5">
            <Label htmlFor="monitor-project">Project</Label>
            {/* A native select: it is accessible by default, works on every
                device, and needs no JavaScript to open. */}
            <select
              id="monitor-project"
              value={effectiveProjectId}
              onChange={(e) => setProjectId(e.target.value)}
              className="h-9 w-full rounded-md border border-input bg-transparent px-3 text-sm focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
            >
              {projects?.map((project) => (
                <option key={project.id} value={project.id}>
                  {project.name}
                </option>
              ))}
            </select>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="monitor-name">Name</Label>
            <Input
              id="monitor-name"
              placeholder="Checkout API"
              aria-invalid={Boolean(errors.name)}
              {...register("name")}
            />
            {errors.name && (
              <p className="text-xs text-status-down">{errors.name.message}</p>
            )}
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="monitor-url">URL</Label>
            <Input
              id="monitor-url"
              placeholder="https://api.example.com/health"
              aria-invalid={Boolean(errors.url)}
              {...register("url")}
            />
            {errors.url && (
              <p className="text-xs text-status-down">{errors.url.message}</p>
            )}
          </div>

          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            <div className="space-y-1.5">
              <Label htmlFor="monitor-method">Method</Label>
              <select
                id="monitor-method"
                className="h-9 w-full rounded-md border border-input bg-transparent px-2 text-sm"
                {...register("method")}
              >
                <option value="GET">GET</option>
                <option value="POST">POST</option>
                <option value="HEAD">HEAD</option>
              </select>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="monitor-interval">Interval (s)</Label>
              <Input
                id="monitor-interval"
                type="number"
                aria-invalid={Boolean(errors.interval_seconds)}
                // valueAsNumber matters: an <input> always yields a STRING, and
                // Zod's .number() would reject "60". This converts on the way in.
                {...register("interval_seconds", { valueAsNumber: true })}
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="monitor-timeout">Timeout (s)</Label>
              <Input
                id="monitor-timeout"
                type="number"
                aria-invalid={Boolean(errors.timeout_seconds)}
                {...register("timeout_seconds", { valueAsNumber: true })}
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="monitor-status">Expect</Label>
              <Input
                id="monitor-status"
                type="number"
                aria-invalid={Boolean(errors.expected_status)}
                {...register("expected_status", { valueAsNumber: true })}
              />
            </div>
          </div>

          {/* The cross-field rule lands on timeout_seconds via the schema's
              `path`, so it appears under the right input. */}
          {(errors.interval_seconds || errors.timeout_seconds || errors.expected_status) && (
            <p className="text-xs text-status-down">
              {errors.timeout_seconds?.message ??
                errors.interval_seconds?.message ??
                errors.expected_status?.message}
            </p>
          )}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={create.isPending}>
              {create.isPending && (
                <Loader2 className="size-4 animate-spin" aria-hidden="true" />
              )}
              Create monitor
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

/** Delete confirmation for a monitor. */
export function DeleteMonitorDialog({ monitor }: { monitor: Monitor }) {
  const [open, setOpen] = useState(false);
  const remove = useDeleteMonitor();

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger
        render={
          <Button variant="ghost" size="sm" aria-label={`Delete ${monitor.name}`} />
        }
      >
        Delete
      </DialogTrigger>

      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <TriangleAlert className="size-4 text-status-down" aria-hidden="true" />
            Delete {monitor.name}?
          </DialogTitle>
          <DialogDescription>
            This permanently deletes the monitor and every check and incident
            recorded for it. This cannot be undone.
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
              remove.mutate(monitor.id, { onSuccess: () => setOpen(false) })
            }
          >
            {remove.isPending && (
              <Loader2 className="size-4 animate-spin" aria-hidden="true" />
            )}
            Delete monitor
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
