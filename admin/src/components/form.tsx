"use client";

import type { ReactNode } from "react";
import { useFormStatus } from "react-dom";

import type { FormState } from "@/lib/form-state";
import { Alert, buttonClass, inputClass } from "./ui";

// Form controls. Inputs are uncontrolled; after an error the action returns
// the submitted values, used as the new default values.

export function SubmitButton({
  children,
  pendingText = "Salvando…",
  variant = "primary",
  className = "",
  pending: actionPending = false,
}: {
  children: ReactNode;
  pendingText?: string;
  variant?: "primary" | "secondary" | "danger" | "ghost";
  className?: string;
  /** Also pending while this is true, e.g. an action dispatched after the form's own action returned. */
  pending?: boolean;
}) {
  const pending = useFormStatus().pending || actionPending;
  return (
    <button type="submit" disabled={pending} className={buttonClass(variant, className)}>
      {pending ? pendingText : children}
    </button>
  );
}

export function FormError({ state }: { state: FormState }) {
  return state.error ? <Alert>{state.error}</Alert> : null;
}

export function Field({
  label,
  name,
  error,
  hint,
  children,
}: {
  label: string;
  name: string;
  error?: string;
  hint?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div>
      <label htmlFor={name} className="block text-sm font-medium text-zinc-800">
        {label}
      </label>
      <div className="mt-1.5">{children}</div>
      {error ? (
        <p id={`${name}-error`} className="mt-1.5 text-sm text-red-600">
          {error}
        </p>
      ) : (
        hint && <p className="mt-1.5 text-xs text-zinc-500">{hint}</p>
      )}
    </div>
  );
}

export function TextInput({
  name,
  error,
  ...props
}: React.InputHTMLAttributes<HTMLInputElement> & { name: string; error?: string }) {
  return (
    <input
      id={name}
      name={name}
      aria-invalid={error ? true : undefined}
      aria-describedby={error ? `${name}-error` : undefined}
      className={inputClass(!!error)}
      {...props}
    />
  );
}

export function Checkbox({
  name,
  label,
  description,
  defaultChecked,
  disabled,
}: {
  name: string;
  label: string;
  description?: string;
  defaultChecked?: boolean;
  disabled?: boolean;
}) {
  return (
    <label className="flex items-start gap-3">
      <input
        type="checkbox"
        name={name}
        defaultChecked={defaultChecked}
        disabled={disabled}
        className="mt-0.5 h-4 w-4 rounded border-zinc-300 accent-indigo-600"
      />
      <span>
        <span className="block text-sm font-medium text-zinc-800">{label}</span>
        {description && <span className="block text-xs text-zinc-500">{description}</span>}
      </span>
    </label>
  );
}
