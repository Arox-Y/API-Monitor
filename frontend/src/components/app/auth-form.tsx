"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import Link from "next/link";
import { Loader2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ApiError } from "@/lib/api";
import {
  loginSchema,
  registerSchema,
  type LoginValues,
  type RegisterValues,
} from "@/lib/schemas";
import { useLogin, useRegister } from "@/hooks/use-auth";

/**
 * The sign-in and sign-up form.
 *
 * One component covers both because they differ only by an extra name field
 * and the wording. Two near-identical files would drift apart.
 */
export function AuthForm({ mode }: { mode: "login" | "register" }) {
  const isRegister = mode === "register";

  const login = useLogin();
  const register = useRegister();
  const mutation = isRegister ? register : login;

  const form = useForm<LoginValues | RegisterValues>({
    // The Zod schema does the validating. The TypeScript types come from the
    // same schema, so the rules and the types can never disagree.
    resolver: zodResolver(isRegister ? registerSchema : loginSchema),
    defaultValues: isRegister
      ? { name: "", email: "", password: "" }
      : { email: "", password: "" },
  });

  const {
    handleSubmit,
    register: field,
    formState: { errors },
    setError,
  } = form;

  function onSubmit(values: LoginValues | RegisterValues) {
    mutation.mutate(values as never, {
      onError: (error) => {
        // The backend validates too, and may reject something the browser
        // allowed. Attach those messages to the right inputs so the user sees
        // them next to the field, not as a vague banner.
        if (error instanceof ApiError && error.details.length > 0) {
          for (const detail of error.details) {
            setError(detail.field as keyof RegisterValues, {
              message: detail.message,
            });
          }
        }
      },
    });
  }

  // A general error worth showing above the form: wrong password, email
  // already taken, server unreachable. Field-level problems are shown inline.
  const generalError =
    mutation.error instanceof ApiError && mutation.error.details.length === 0
      ? mutation.error.message
      : null;

  return (
    <div className="rounded-lg border border-border p-6">
      <h1 className="text-lg font-semibold tracking-tight">
        {isRegister ? "Create an account" : "Sign in"}
      </h1>
      <p className="mt-1 text-sm text-muted-foreground">
        {isRegister
          ? "Start monitoring your APIs in a minute."
          : "Welcome back."}
      </p>

      <form onSubmit={handleSubmit(onSubmit)} className="mt-6 space-y-4" noValidate>
        {generalError && (
          // role="alert" makes a screen reader announce this immediately,
          // rather than the user wondering why nothing happened.
          <div
            role="alert"
            className="rounded-md border border-status-down/25 bg-status-down/10 px-3 py-2 text-sm text-status-down"
          >
            {generalError}
          </div>
        )}

        {isRegister && (
          <Field
            id="name"
            label="Name"
            error={"name" in errors ? errors.name?.message : undefined}
          >
            <Input
              id="name"
              autoComplete="name"
              // aria-invalid tells assistive tech the field is wrong; the
              // red border alone conveys nothing to a screen reader.
              aria-invalid={"name" in errors && Boolean(errors.name)}
              {...field("name" as keyof RegisterValues)}
            />
          </Field>
        )}

        <Field id="email" label="Email" error={errors.email?.message}>
          <Input
            id="email"
            type="email"
            autoComplete="email"
            aria-invalid={Boolean(errors.email)}
            {...field("email")}
          />
        </Field>

        <Field id="password" label="Password" error={errors.password?.message}>
          <Input
            id="password"
            type="password"
            // Tells password managers whether to offer a saved password or
            // generate a new one.
            autoComplete={isRegister ? "new-password" : "current-password"}
            aria-invalid={Boolean(errors.password)}
            {...field("password")}
          />
        </Field>

        <Button type="submit" className="w-full" disabled={mutation.isPending}>
          {mutation.isPending && (
            <Loader2 className="size-4 animate-spin" aria-hidden="true" />
          )}
          {mutation.isPending
            ? isRegister
              ? "Creating account..."
              : "Signing in..."
            : isRegister
              ? "Create account"
              : "Sign in"}
        </Button>
      </form>

      <p className="mt-6 text-sm text-muted-foreground">
        {isRegister ? "Already have an account? " : "No account? "}
        <Link
          href={isRegister ? "/login" : "/register"}
          className="text-foreground hover:underline"
        >
          {isRegister ? "Sign in" : "Create one"}
        </Link>
      </p>
    </div>
  );
}

/** A label, an input and its error message, wired together for accessibility. */
function Field({
  id,
  label,
  error,
  children,
}: {
  id: string;
  label: string;
  error?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="space-y-1.5">
      {/* htmlFor links the label to the input, so clicking the label focuses
          the field and screen readers announce the two together. */}
      <Label htmlFor={id}>{label}</Label>
      {children}
      {error && (
        <p id={`${id}-error`} className="text-xs text-status-down">
          {error}
        </p>
      )}
    </div>
  );
}
