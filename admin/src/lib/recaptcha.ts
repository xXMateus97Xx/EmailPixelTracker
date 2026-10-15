import "server-only";

import { unstable_rethrow } from "next/navigation";

import { api } from "./api";
import type { RecaptchaConfig, RecaptchaSettings } from "./types";

/**
 * Whether the login and the password change need a reCAPTCHA v3 token, as the
 * API says. If the API cannot be reached the form is shown without it; the
 * request then fails with the connection error anyway.
 */
export async function getRecaptchaConfig(): Promise<RecaptchaConfig> {
  try {
    return await api<RecaptchaConfig>("GET", "/api/auth/recaptcha", { anonymous: true });
  } catch (err) {
    unstable_rethrow(err);
    console.error("reading the reCAPTCHA configuration", err);
    return { enabled: false };
  }
}

/** The settings a form passes to the browser, or undefined when reCAPTCHA is off. */
export async function recaptchaFor(form: "login" | "changePassword"): Promise<RecaptchaSettings | undefined> {
  const config = await getRecaptchaConfig();
  if (!config.enabled || !config.siteKey) {
    return undefined;
  }
  const action = form === "login" ? (config.action ?? "login") : (config.changePasswordAction ?? "change_password");
  return { siteKey: config.siteKey, action };
}
