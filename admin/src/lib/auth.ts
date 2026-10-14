import "server-only";

import { redirect } from "next/navigation";
import { cache } from "react";

import { api, ApiError } from "./api";
import type { Me } from "./types";

/**
 * The signed-in user, its tenant and permissions, loaded once per request.
 * Sends the visitor to the login page when the session is missing, expired or
 * no longer valid (user or tenant deactivated).
 */
export const getMe = cache(async (): Promise<Me> => {
  try {
    return await api<Me>("GET", "/api/auth/me");
  } catch (err) {
    if (err instanceof ApiError && err.status === 401) {
      redirect("/login?reason=expired");
    }
    if (err instanceof ApiError && err.status === 403) {
      redirect("/login?reason=tenant-inactive");
    }
    throw err;
  }
});

/**
 * Stops pages the user cannot use. Hiding links is only a convenience: the
 * API enforces every rule again.
 */
export async function requirePermission(allowed: (me: Me) => boolean): Promise<Me> {
  const me = await getMe();
  if (!allowed(me)) {
    redirect("/");
  }
  return me;
}
