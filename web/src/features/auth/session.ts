import { useQuery } from "@tanstack/react-query";

import { ApiError, api } from "../../shared/api/client";

export const sessionQueryKey = ["session"] as const;

/**
 * Resolves the current session from the server. The browser holds an opaque
 * cookie it cannot read, so the only way to know whether a session is valid is
 * to ask. An expired session is not an error state for the UI: it means the
 * person needs to sign in again.
 */
export function useSession() {
  return useQuery({
    queryKey: sessionQueryKey,
    queryFn: () => api.session(),
    retry: (failureCount, error) => {
      if (error instanceof ApiError && error.isUnauthenticated) return false;
      return failureCount < 2;
    },
    staleTime: 60_000,
  });
}

export function isUnauthenticated(error: unknown): boolean {
  return error instanceof ApiError && error.isUnauthenticated;
}
