"use server";

import { redirect } from "next/navigation";

import { api, ApiError } from "@/lib/api";
import type { FormState } from "@/lib/form-state";
import { errorState, formValues } from "@/lib/forms";
import { setToken } from "@/lib/session";
import type { Token } from "@/lib/types";

export async function login(_: FormState, formData: FormData): Promise<FormState> {
  const { recaptchaToken, ...values } = formValues(formData);
  try {
    const res = await api<Token>("POST", "/api/auth/login", {
      anonymous: true,
      body: {
        username: values.username ?? "",
        password: String(formData.get("password") ?? ""),
        // Sent only when the form made one (reCAPTCHA enabled).
        recaptchaToken: recaptchaToken || undefined,
      },
    });
    await setToken(res.token, res.expiresAtUtc);
  } catch (err) {
    // The token is not a field of the form: reCAPTCHA was enabled after the
    // page loaded, or JavaScript is off.
    if (err instanceof ApiError && err.fields.some((f) => f.field === "recaptchaToken")) {
      return { error: "O login agora exige reCAPTCHA. Recarregue a página (com JavaScript ativo) e tente novamente.", values };
    }
    return errorState(err, values);
  }
  redirect("/");
}
