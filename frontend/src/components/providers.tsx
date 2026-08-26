"use client";

import { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { ApiError } from "@/lib/api";

/**
 * Wraps the app in TanStack Query, which manages all server data.
 *
 * Server data is different from normal React state: it lives somewhere else,
 * it can change without us doing anything, and it can be stale. Trying to
 * manage it with useState and useEffect means hand-writing loading flags,
 * error flags, race-condition handling and caching - which is this library,
 * rebuilt badly.
 */
export function Providers({ children }: { children: React.ReactNode }) {
  // The client is created inside useState, not at module level.
  //
  // A module-level client would be shared by every user on the server during
  // server rendering, leaking one person's cached data to another. useState
  // gives each browser session its own.
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            // How long data is considered fresh before a refetch is allowed.
            staleTime: 30_000,

            retry: (failureCount, error) => {
              // Never retry a 401. The user is not signed in, and asking again
              // will not change that - it just delays the redirect.
              if (error instanceof ApiError && error.isUnauthorized) {
                return false;
              }
              // Nor a 404: the thing does not exist.
              if (error instanceof ApiError && error.status === 404) {
                return false;
              }
              return failureCount < 2;
            },

            // Refetching whenever the tab regains focus is useful for a
            // monitoring dashboard: come back to it and the numbers are
            // current rather than however old the tab is.
            refetchOnWindowFocus: true,
          },
          mutations: {
            // A mutation changes data. Retrying automatically could create the
            // same project twice, so failures surface to the user instead.
            retry: false,
          },
        },
      }),
  );

  return (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}
