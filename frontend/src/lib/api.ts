/**
 * The single place this app talks to the backend.
 *
 * Every request goes through here, which means the rules below are written
 * once instead of being repeated - and forgotten - in each component.
 */

/** Where the Go backend lives. NEXT_PUBLIC_ means it is safe for the browser. */
const API_BASE =
  process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

/** One field-level problem, matching the backend's error envelope. */
export type FieldError = {
  field: string;
  message: string;
};

/**
 * An error the API returned deliberately, as opposed to the network failing.
 *
 * Extending Error means it can be thrown and caught like any other error,
 * while carrying the extra information the UI needs.
 */
export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
    public readonly details: FieldError[] = [],
  ) {
    super(message);
    this.name = "ApiError";
  }

  /** True when the user simply is not signed in. */
  get isUnauthorized(): boolean {
    return this.status === 401;
  }

  /** Field errors keyed by field name, ready to hand to a form. */
  get fieldErrors(): Record<string, string> {
    return Object.fromEntries(
      this.details.map((detail) => [detail.field, detail.message]),
    );
  }
}

/** The shape the backend uses for every error. See docs/API_SPEC.md §1. */
type ErrorEnvelope = {
  error: {
    code: string;
    message: string;
    details?: FieldError[];
  };
};

/**
 * Make a request to the API.
 *
 * Returns the parsed body on success, and throws ApiError on failure.
 * Throwing rather than returning an error object is deliberate: TanStack Query
 * treats a thrown error as a failed query, which is what drives the automatic
 * error states in the UI.
 */
export async function apiFetch<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  let response: Response;

  try {
    response = await fetch(`${API_BASE}${path}`, {
      ...options,

      // THE CRITICAL LINE.
      //
      // Our session lives in an httpOnly cookie. Browsers do NOT send cookies
      // on cross-origin requests unless asked, and localhost:3000 calling
      // localhost:8080 is cross-origin. Without this, every request looks
      // logged-out and nothing works.
      credentials: "include",

      headers: {
        "Content-Type": "application/json",
        ...options.headers,
      },
    });
  } catch {
    // fetch only rejects when the request never completed: the server is
    // down, DNS failed, the connection dropped. An HTTP 500 is a successful
    // request with a bad status, and lands below instead.
    throw new ApiError(
      0,
      "NETWORK_ERROR",
      "Could not reach the server. Is the backend running?",
    );
  }

  // 204 No Content has no body to parse - calling .json() on it would throw.
  if (response.status === 204) {
    return undefined as T;
  }

  const text = await response.text();

  if (!response.ok) {
    // Try to read our standard error envelope. A crashed server or a proxy
    // might return HTML instead, so this must not itself explode.
    try {
      const parsed = JSON.parse(text) as ErrorEnvelope;
      throw new ApiError(
        response.status,
        parsed.error.code,
        parsed.error.message,
        parsed.error.details ?? [],
      );
    } catch (error) {
      if (error instanceof ApiError) throw error;
      throw new ApiError(
        response.status,
        "UNKNOWN_ERROR",
        `Request failed with status ${response.status}.`,
      );
    }
  }

  return JSON.parse(text) as T;
}

/** Convenience wrappers so callers do not repeat method and JSON.stringify. */
export const api = {
  get: <T>(path: string) => apiFetch<T>(path, { method: "GET" }),

  post: <T>(path: string, body?: unknown) =>
    apiFetch<T>(path, {
      method: "POST",
      body: body === undefined ? undefined : JSON.stringify(body),
    }),

  patch: <T>(path: string, body: unknown) =>
    apiFetch<T>(path, { method: "PATCH", body: JSON.stringify(body) }),

  delete: <T>(path: string) => apiFetch<T>(path, { method: "DELETE" }),
};
