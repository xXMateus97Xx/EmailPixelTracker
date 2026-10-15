"use client";

import { useActionState, useEffect, useRef } from "react";

import { Field, FormError, SubmitButton, TextInput } from "@/components/form";
import { useRecaptcha } from "@/components/recaptcha";
import { Alert, Card } from "@/components/ui";
import { emptyForm } from "@/lib/form-state";
import type { RecaptchaSettings } from "@/lib/types";
import { changePassword } from "./actions";

export function PasswordForm({ username, recaptcha }: { username: string; recaptcha?: RecaptchaSettings }) {
  const [state, dispatch, isPending] = useActionState(changePassword, emptyForm);
  const captcha = useRecaptcha(recaptcha, dispatch);
  const form = useRef<HTMLFormElement>(null);
  const f = state.fields ?? {};

  // React clears the fields after a plain form action, but not after the
  // dispatch made by the reCAPTCHA submit.
  useEffect(() => {
    if (state.success) {
      form.current?.reset();
    }
  }, [state]);

  return (
    <Card className="max-w-xl">
      <form ref={form} action={dispatch} onSubmit={captcha.onSubmit} className="space-y-5 p-6">
        {captcha.script}
        {captcha.error ? <Alert>{captcha.error}</Alert> : <FormError state={state} />}
        {state.success && !captcha.error && <Alert tone="green">{state.success}</Alert>}
        {/* Tells password managers whose password this is. */}
        <input type="text" name="username" value={username} autoComplete="username" readOnly hidden />
        <Field label="Senha atual" name="currentPassword" error={f.currentPassword}>
          <TextInput name="currentPassword" type="password" autoComplete="current-password" required error={f.currentPassword} />
        </Field>
        <Field label="Nova senha" name="newPassword" error={f.newPassword} hint="Pelo menos 8 caracteres.">
          <TextInput name="newPassword" type="password" autoComplete="new-password" required minLength={8} error={f.newPassword} />
        </Field>
        <div className="pt-2">
          <SubmitButton pending={captcha.gettingToken || isPending}>Alterar senha</SubmitButton>
        </div>
      </form>
    </Card>
  );
}
