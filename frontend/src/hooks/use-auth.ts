"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";

import { api } from "@/lib/api";
import type { LoginValues, RegisterValues } from "@/lib/schemas";

/** The user object the API returns. Never includes a password hash. */
export type AuthUser = {
  id: string;
  email: string;
  name: string;
  created_at: string;
  updated_at: string;
};

/**
 * Query keys, defined in one place.
 *
 * Keys identify cached data. Keeping them here rather than typing the strings
 * inline prevents the classic bug where one file caches under ["auth","me"]
 * and another invalidates ["auth", "user"] - so the screen never updates.
 */
export const authKeys = {
  me: ["auth", "me"] as const,
};

/**
 * The currently signed-in user, or null.
 *
 * A 401 here is a normal answer ("nobody is signed in"), not a failure, so it
 * is turned into null rather than an error state.
 */
export function useMe() {
  return useQuery({
    queryKey: authKeys.me,
    queryFn: async () => {
      try {
        return await api.get<AuthUser>("/api/v1/auth/me");
      } catch {
        return null;
      }
    },
  });
}

export function useLogin() {
  const queryClient = useQueryClient();
  const router = useRouter();

  return useMutation({
    mutationFn: (values: LoginValues) =>
      api.post<AuthUser>("/api/v1/auth/login", values),

    onSuccess: (user) => {
      // Write the user straight into the cache so the dashboard does not have
      // to fetch it again immediately after redirecting.
      queryClient.setQueryData(authKeys.me, user);
      router.push("/dashboard");
    },
  });
}

export function useRegister() {
  const queryClient = useQueryClient();
  const router = useRouter();

  return useMutation({
    mutationFn: (values: RegisterValues) =>
      api.post<AuthUser>("/api/v1/auth/register", values),

    onSuccess: (user) => {
      queryClient.setQueryData(authKeys.me, user);
      router.push("/dashboard");
    },
  });
}

export function useLogout() {
  const queryClient = useQueryClient();
  const router = useRouter();

  return useMutation({
    mutationFn: () => api.post<void>("/api/v1/auth/logout"),

    onSuccess: () => {
      // Clear EVERYTHING, not just the user.
      //
      // The cache still holds the previous user's projects and monitors. If
      // somebody else signs in on this browser, they would briefly see the
      // last person's data before the refetch completes.
      queryClient.clear();
      router.push("/login");
    },
  });
}
