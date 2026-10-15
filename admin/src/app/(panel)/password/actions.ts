"use server";

import { api, ApiError } from "@/lib/api";
import type { FormState } from "@/lib/form-state";
import { errorState, formValues } from "@/lib/forms";

export async function changePassword(_: FormState, formData: FormData): Promise<FormState> {
  const { recaptchaToken } = formValues(formData);
  try {
    await api("POST", "/api/auth/change-password", {
      body: {
        currentPassword: String(formData.get("currentPassword") ?? ""),
        newPassword: String(formData.get("newPassword") ?? ""),
        // Sent only when the form made one (reCAPTCHA enabled).
        recaptchaToken: recaptchaToken || undefined,
      },
    });
  } catch (err) {
    // reCAPTCHA was enabled after the page loaded, or JavaScript is off.
    if (err instanceof ApiError && err.fields.some((f) => f.field === "recaptchaToken")) {
      return { error: "A troca de senha agora exige reCAPTCHA. Recarregue a página (com JavaScript ativo) e tente novamente." };
    }
    // No values to keep: the fields are all passwords.
    return errorState(err, {});
  }
  return { success: "Senha alterada. Use a nova senha no próximo login." };
}
