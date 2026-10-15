import "server-only";

import { redirect } from "next/navigation";

import { ApiError } from "./api";
import type { FormState } from "./form-state";

export function formValues(formData: FormData): Record<string, string> {
  const values: Record<string, string> = {};
  for (const [key, value] of formData.entries()) {
    // Passwords (password, currentPassword, newPassword) are never sent back to the browser.
    if (typeof value === "string" && !key.startsWith("$") && !/password/i.test(key)) {
      values[key] = value;
    }
  }
  return values;
}

// The API answers in English; the panel speaks Portuguese.
const exact: Record<string, string> = {
  "is already in use": "já está em uso",
  "is required": "é obrigatório",
  "must not contain spaces": "não pode conter espaços",
  "must be in the future": "precisa ser uma data futura",
  "must be after the campaign creation date": "precisa ser posterior à criação da campanha",
  "you cannot deactivate your own user": "você não pode desativar o próprio usuário",
  "must have at most 72 bytes": "pode ter no máximo 72 bytes",
  "is incorrect": "está incorreta",
  "must be different from the current password": "precisa ser diferente da senha atual",
  "must be after fromUtc": "precisa ser posterior à data inicial",
  "invalid credentials": "Usuário ou senha inválidos.",
  "tenant is inactive": "O tenant está inativo.",
  forbidden: "Você não tem permissão para esta ação.",
  "not found": "Registro não encontrado.",
  conflict: "Outro cadastro com os mesmos dados foi salvo ao mesmo tempo. Tente novamente.",
  "impersonation required to access tenant data": "Acesse um tenant para continuar.",
  "recaptcha verification failed": "Não foi possível confirmar que você não é um robô. Tente novamente.",
  "recaptcha verification unavailable": "Não foi possível verificar o reCAPTCHA agora. Tente novamente em instantes.",
};

const patterns: [RegExp, (m: RegExpMatchArray) => string][] = [
  [/^must have between (\d+) and (\d+) characters$/, (m) => `deve ter entre ${m[1]} e ${m[2]} caracteres`],
  [/^must have at least (\d+) characters$/, (m) => `deve ter pelo menos ${m[1]} caracteres`],
  [/^must be between (\S+) and (\S+)$/, (m) => `deve estar entre ${m[1]} e ${m[2]}`],
  [/^must be one of (.+)$/, (m) => `deve ser uma destas: ${m[1].replace(" or ", ", ")}`],
  [/impersonated tenant is read-only/, () => "O tenant está inativo: o acesso é somente leitura."],
];

export function translate(message: string): string {
  if (exact[message]) {
    return exact[message];
  }
  for (const [pattern, format] of patterns) {
    const match = message.match(pattern);
    if (match) {
      return format(match);
    }
  }
  return message;
}

/** Turns an error of a Server Action into the state shown by the form. */
export function errorState(err: unknown, values: Record<string, string>): FormState {
  if (!(err instanceof ApiError)) {
    throw err; // redirect() and unexpected errors propagate
  }
  // The session expired or the user was deactivated while the form was open.
  if (err.status === 401 && err.message !== "invalid credentials") {
    redirect("/login?reason=expired");
  }
  if (err.fields.length > 0) {
    const fields: Record<string, string> = {};
    for (const f of err.fields) {
      fields[f.field] ??= translate(f.message);
    }
    return { error: "Corrija os campos destacados.", fields, values };
  }
  let error = translate(err.message);
  // Known errors (e.g. 503 from reCAPTCHA) keep their translated message.
  if (err.status >= 500 && !exact[err.message]) {
    error = "Erro interno da API." + (err.requestId ? ` Código para suporte: ${err.requestId}` : "");
  }
  return { error, values };
}
