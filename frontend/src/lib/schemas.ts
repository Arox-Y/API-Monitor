import { z } from "zod";

/**
 * Form validation rules.
 *
 * Each schema is the SINGLE source of truth: it validates at runtime, and the
 * TypeScript type is inferred from it with z.infer. Writing a matching
 * `interface` by hand would create two things that must be kept in sync, and
 * they always drift apart eventually.
 *
 * IMPORTANT: these rules exist for the USER's benefit - instant feedback
 * without a round trip. They are not security. Anyone can bypass the browser
 * entirely, which is why the Go backend re-validates everything. Client
 * validation is UX; server validation is safety.
 *
 * The rules below mirror the backend exactly (docs/API_SPEC.md §3).
 */

export const loginSchema = z.object({
  email: z.email("Enter a valid email address"),
  password: z.string().min(1, "Password is required"),
});

export type LoginValues = z.infer<typeof loginSchema>;

export const registerSchema = z.object({
  name: z
    .string()
    .min(1, "Name is required")
    .max(100, "Name must be at most 100 characters"),
  email: z
    .email("Enter a valid email address")
    .max(255, "Email must be at most 255 characters"),
  password: z
    .string()
    // Matches the backend's policy exactly, so the form never accepts
    // something the server will reject.
    .min(8, "Password must be at least 8 characters")
    // 72 is bcrypt's hard limit: it silently ignores anything beyond that, so
    // we reject rather than give a false sense of a longer password.
    .max(72, "Password must be at most 72 characters"),
});

export type RegisterValues = z.infer<typeof registerSchema>;

export const projectSchema = z.object({
  name: z
    .string()
    .min(1, "Name is required")
    .max(100, "Name must be at most 100 characters"),
  description: z
    .string()
    .max(500, "Description must be at most 500 characters")
    .optional(),
  is_public: z.boolean().optional(),
});

export type ProjectValues = z.infer<typeof projectSchema>;

export const monitorSchema = z
  .object({
    name: z
      .string()
      .min(1, "Name is required")
      .max(100, "Name must be at most 100 characters"),
    url: z
      .string()
      .min(1, "URL is required")
      .refine(
        (value) => /^https?:\/\/.+/i.test(value),
        "URL must start with http:// or https://",
      ),
    method: z.enum(["GET", "POST", "HEAD"]),
    interval_seconds: z
      .number()
      .int()
      .min(30, "Minimum interval is 30 seconds")
      .max(86400, "Maximum interval is 24 hours"),
    timeout_seconds: z
      .number()
      .int()
      .min(1, "Minimum timeout is 1 second")
      .max(30, "Maximum timeout is 30 seconds"),
    expected_status: z
      .number()
      .int()
      .min(100, "Must be a valid HTTP status code")
      .max(599, "Must be a valid HTTP status code"),
    enabled: z.boolean(),
  })
  // CROSS-FIELD RULE.
  //
  // Every field above can be individually valid while the combination is
  // nonsense: a 30-second timeout on a 30-second interval means the next
  // check begins before the previous one has given up.
  //
  // .refine() runs after the individual field checks and sees the whole
  // object, which is the only way to express a rule spanning two fields.
  // `path` attaches the message to the timeout input rather than the form.
  .refine((data) => data.timeout_seconds < data.interval_seconds, {
    message: "Timeout must be shorter than the check interval",
    path: ["timeout_seconds"],
  });

export type MonitorValues = z.infer<typeof monitorSchema>;
