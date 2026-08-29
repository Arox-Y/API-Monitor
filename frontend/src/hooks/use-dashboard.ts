"use client";

import { useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";
import type { DashboardStats, MonitorDetail } from "@/types/api";

export const dashboardKeys = {
  stats: ["dashboard", "stats"] as const,
  monitorDetail: (id: string) => ["dashboard", "monitor", id] as const,
};

/**
 * The whole dashboard in one request.
 *
 * One aggregate endpoint rather than five REST calls means one loading state
 * and one failure mode on this screen instead of five (ADR-005).
 */
export function useDashboardStats() {
  return useQuery({
    queryKey: dashboardKeys.stats,
    queryFn: () => api.get<DashboardStats>("/api/v1/dashboard/stats"),
    // Monitors change on their own while you are looking at the page, so poll.
    // This is why WebSockets are not needed for the MVP.
    refetchInterval: 30_000,
  });
}

/** One monitor with its checks, incidents and uptime. */
export function useMonitorDetail(id: string) {
  return useQuery({
    queryKey: dashboardKeys.monitorDetail(id),
    queryFn: () => api.get<MonitorDetail>(`/api/v1/monitors/${id}/checks?limit=100`),
    enabled: Boolean(id),
    refetchInterval: 30_000,
  });
}
