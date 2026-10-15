"use client";

import { useActionState } from "react";

import { Field, FormError, SubmitButton, TextInput } from "@/components/form";
import { useRecaptcha } from "@/components/recaptcha";
import { Alert } from "@/components/ui";
import { emptyForm } from "@/lib/form-state";
import type { RecaptchaSettings } from "@/lib/types";
import { login } from "./actions";

export function LoginForm({ recaptcha }: { recaptcha?: RecaptchaSettings }) {
  const [state, dispatch, isPending] = useActionState(login, emptyForm);
  const captcha = useRecaptcha(recaptcha, dispatch);

  return (
    <form action={dispatch} onSubmit={captcha.onSubmit} className="space-y-5">
      {captcha.script}
      {captcha.error ? <Alert>{captcha.error}</Alert> : <FormError state={state} />}
      <Field label="Usuário" name="username">
        <TextInput
          name="username"
          autoComplete="username"
          autoCapitalize="none"
          spellCheck={false}
          required
          autoFocus
          defaultValue={state.values?.username}
        />
      </Field>
      <Field label="Senha" name="password">
        <TextInput name="password" type="password" autoComplete="current-password" required />
      </Field>
      <SubmitButton pendingText="Entrando…" pending={captcha.gettingToken || isPending} className="w-full">
        Entrar
      </SubmitButton>
    </form>
  );
}
