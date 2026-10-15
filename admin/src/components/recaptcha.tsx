"use client";

import Script from "next/script";
import { startTransition, useState } from "react";

import type { RecaptchaSettings } from "@/lib/types";
import { Alert } from "./ui";

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
function recaptchaToken({ siteKey, action }: RecaptchaSettings): Promise<string> {
  return new Promise((resolve, reject) => {
    const grecaptcha = window.grecaptcha;
    if (!grecaptcha) {
      reject(new Error("reCAPTCHA not loaded"));
      return;
    }
    grecaptcha.ready(() => grecaptcha.execute(siteKey, { action }).then(resolve, reject));
  });
}

/**
 * Sends the form to dispatch with a reCAPTCHA token in "recaptchaToken".
 *
 * Without reCAPTCHA, onSubmit is undefined and the form posts straight to the
 * Server Action (it works even before hydration). With it, the submit waits
 * for a token and then dispatches; preventing the default submit also keeps
 * React from resetting the fields while the action is still running.
 */
export function useRecaptcha(recaptcha: RecaptchaSettings | undefined, dispatch: (formData: FormData) => void) {
  const [error, setError] = useState<string>();
  const [gettingToken, setGettingToken] = useState(false);

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    setError(undefined);
    setGettingToken(true);
    try {
      formData.set("recaptchaToken", await recaptchaToken(recaptcha!));
    } catch (err) {
      console.error(err);
      setError(LOAD_ERROR);
      return;
    } finally {
      setGettingToken(false);
    }
    startTransition(() => dispatch(formData));
  }

  return {
    onSubmit: recaptcha ? submit : undefined,
    /** Shown instead of the form's error: the token could not be made. */
    error,
    gettingToken,
    script: recaptcha && (
      <>
        <Script
          src={`https://www.google.com/recaptcha/api.js?render=${encodeURIComponent(recaptcha.siteKey)}`}
          strategy="afterInteractive"
          onError={() => setError(LOAD_ERROR)}
        />
        <noscript>
          <Alert>Este formulário usa reCAPTCHA: ative o JavaScript para continuar.</Alert>
        </noscript>
      </>
    ),
  };
}
