"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/lib/api";
import type { ProjectValues } from "@/lib/schemas";
import type { Project } from "@/types/api";

/**
 * Query keys are HIERARCHICAL, and that hierarchy is the point.
 *
 *   ["projects"]              all projects
 *   ["projects", "abc-123"]   one project
 *
 * Invalidating ["projects"] also invalidates everything nested beneath it, so
 * after creating a project one call refreshes both the list and any detail
 * views without naming them individually.
 */
export const projectKeys = {
  all: ["projects"] as const,
  detail: (id: string) => ["projects", id] as const,
};

type ProjectListResponse = { projects: Project[] };

export function useProjects() {
  return useQuery({
    queryKey: projectKeys.all,
    queryFn: async () => {
      const data = await api.get<ProjectListResponse>("/api/v1/projects");
      return data.projects;
    },
  });
}

export function useProject(id: string) {
  return useQuery({
    queryKey: projectKeys.detail(id),
    queryFn: () => api.get<Project>(`/api/v1/projects/${id}`),
    // Do not fire the request until an id actually exists.
    enabled: Boolean(id),
  });
}

export function useCreateProject() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (values: ProjectValues) =>
      api.post<Project>("/api/v1/projects", values),

    // After changing data, INVALIDATE rather than hand-editing the cache.
    //
    // Patching the cache by hand is faster but easy to get wrong: the server
    // also generated a slug and a monitor_count we did not have. Invalidating
    // refetches the truth. Correctness first; optimise only if it feels slow.
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: projectKeys.all });
    },
  });
}

export function useUpdateProject(id: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (values: Partial<ProjectValues>) =>
      api.patch<Project>(`/api/v1/projects/${id}`, values),

    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: projectKeys.all });
    },
  });
}

export function useDeleteProject() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.delete<void>(`/api/v1/projects/${id}`),

    onSuccess: () => {
      // Deleting a project cascades to its monitors in the database, so the
      // monitor list is stale too. Both caches must be refreshed or the UI
      // will show monitors that no longer exist.
      queryClient.invalidateQueries({ queryKey: projectKeys.all });
      queryClient.invalidateQueries({ queryKey: ["monitors"] });
    },
  });
}
