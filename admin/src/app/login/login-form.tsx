"use client";

import Script from "next/script";
import { startTransition, useActionState, useState } from "react";

import { Field, FormError, SubmitButton, TextInput } from "@/components/form";
import { Alert } from "@/components/ui";
import { emptyForm } from "@/lib/form-state";
import { login } from "./actions";

type Recaptcha = { siteKey: string; action: string };

const LOAD_ERROR = "Não foi possível carregar o reCAPTCHA. Verifique sua conexão e recarregue a página.";

declare global {
  interface Window {
    grecaptcha?: {
      ready(callback: () => void): void;
      execute(siteKey: string, options: { action: string }): Promise<string>;
    };
  }
}

/** A fresh token for each attempt: they expire after 2 minutes and work once. */
function recaptchaToken({ siteKey, action }: Recaptcha): Promise<string> {
  return new Promise((resolve, reject) => {
    const grecaptcha = window.grecaptcha;
    if (!grecaptcha) {
      reject(new Error("reCAPTCHA not loaded"));
      return;
    }
    grecaptcha.ready(() => grecaptcha.execute(siteKey, { action }).then(resolve, reject));
  });
}

export function LoginForm({ recaptcha }: { recaptcha?: Recaptcha }) {
  const [state, dispatch, isPending] = useActionState(login, emptyForm);
  const [recaptchaError, setRecaptchaError] = useState<string>();
  const [gettingToken, setGettingToken] = useState(false);

  // Without reCAPTCHA the form posts straight to the Server Action (it works
  // even before hydration). With it, the submit waits for a token and then
  // dispatches; preventing the default submit also keeps React from resetting
  // the fields while the login is still running.
  async function submitWithRecaptcha(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    setRecaptchaError(undefined);
    setGettingToken(true);
    try {
      formData.set("recaptchaToken", await recaptchaToken(recaptcha!));
    } catch (err) {
      console.error(err);
      setRecaptchaError(LOAD_ERROR);
      return;
    } finally {
      setGettingToken(false);
    }
    startTransition(() => dispatch(formData));
  }

  return (
    <form action={dispatch} onSubmit={recaptcha ? submitWithRecaptcha : undefined} className="space-y-5">
      {recaptcha && (
        <>
          <Script
            src={`https://www.google.com/recaptcha/api.js?render=${encodeURIComponent(recaptcha.siteKey)}`}
            strategy="afterInteractive"
            onError={() => setRecaptchaError(LOAD_ERROR)}
          />
          <noscript>
            <Alert>O login usa reCAPTCHA: ative o JavaScript para entrar.</Alert>
          </noscript>
        </>
      )}
      {recaptchaError ? <Alert>{recaptchaError}</Alert> : <FormError state={state} />}
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
      <SubmitButton pendingText="Entrando…" pending={gettingToken || isPending} className="w-full">
        Entrar
      </SubmitButton>
    </form>
  );
}
