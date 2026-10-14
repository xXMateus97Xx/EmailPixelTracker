import "server-only";

import { api } from "./api";
import type { RecaptchaConfig } from "./types";

/**
 * Whether the login needs a reCAPTCHA v3 token, as the API says. If the API
 * cannot be reached the form is shown without it; the login then fails with
 * the connection error anyway.
 */
export async function getRecaptchaConfig(): Promise<RecaptchaConfig> {
  try {
    return await api<RecaptchaConfig>("GET", "/api/auth/recaptcha", { anonymous: true });
  } catch (err) {
    console.error("reading the reCAPTCHA configuration", err);
    return { enabled: false };
  }
}
