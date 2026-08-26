"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/lib/api";
import type { MonitorValues } from "@/lib/schemas";
import type { Monitor } from "@/types/api";

export const monitorKeys = {
  all: ["monitors"] as const,
  detail: (id: string) => ["monitors", id] as const,
  byProject: (projectId: string) => ["monitors", "project", projectId] as const,
};

type MonitorListResponse = { monitors: Monitor[] };

/** Every monitor the signed-in user owns. */
export function useMonitors() {
  return useQuery({
    queryKey: monitorKeys.all,
    queryFn: async () => {
      const data = await api.get<MonitorListResponse>("/api/v1/monitors");
      return data.monitors;
    },
    // Monitors go up and down on their own, so poll.
    //
    // This is why WebSockets are NOT needed for the MVP: a 30-second refresh
    // is close enough to live for a monitoring dashboard, and costs no extra
    // infrastructure. WebSockets stay an optional stretch goal.
    refetchInterval: 30_000,
  });
}

export function useProjectMonitors(projectId: string) {
  return useQuery({
    queryKey: monitorKeys.byProject(projectId),
    queryFn: async () => {
      const data = await api.get<MonitorListResponse>(
        `/api/v1/projects/${projectId}/monitors`,
      );
      return data.monitors;
    },
    enabled: Boolean(projectId),
  });
}

export function useMonitor(id: string) {
  return useQuery({
    queryKey: monitorKeys.detail(id),
    queryFn: () => api.get<Monitor>(`/api/v1/monitors/${id}`),
    enabled: Boolean(id),
  });
}

export function useCreateMonitor(projectId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (values: MonitorValues) =>
      api.post<Monitor>(`/api/v1/projects/${projectId}/monitors`, values),

    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: monitorKeys.all });
      // The project's monitor_count changed too.
      queryClient.invalidateQueries({ queryKey: ["projects"] });
    },
  });
}

export function useUpdateMonitor(id: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (values: Partial<MonitorValues>) =>
      api.patch<Monitor>(`/api/v1/monitors/${id}`, values),

    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: monitorKeys.all });
      queryClient.invalidateQueries({ queryKey: monitorKeys.detail(id) });
    },
  });
}

/**
 * Enable or disable a monitor.
 *
 * This is just an update with one field, not a separate endpoint. `enabled` is
 * a field, and PATCH is the verb for changing fields - inventing /pause would
 * be a second way to do the same thing.
 */
export function useToggleMonitor() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      api.patch<Monitor>(`/api/v1/monitors/${id}`, { enabled }),

    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: monitorKeys.all });
    },
  });
}

export function useDeleteMonitor() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.delete<void>(`/api/v1/monitors/${id}`),

    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: monitorKeys.all });
      queryClient.invalidateQueries({ queryKey: ["projects"] });
    },
  });
}
