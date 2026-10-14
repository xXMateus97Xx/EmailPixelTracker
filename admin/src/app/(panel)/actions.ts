"use server";

import { redirect } from "next/navigation";

import { api, ApiError } from "@/lib/api";
import { clearToken, setToken } from "@/lib/session";
import type { Token } from "@/lib/types";

export async function logout(): Promise<void> {
  await clearToken();
  redirect("/login");
}

/** Exchanges the SuperAdmin token for one scoped to the tenant. */
export async function impersonate(tenantId: string): Promise<void> {
  await switchToken("/api/auth/impersonate", { tenantId });
  redirect("/campaigns");
}

export async function stopImpersonation(): Promise<void> {
  await switchToken("/api/auth/stop-impersonation");
  redirect("/tenants");
}

async function switchToken(path: string, body?: unknown): Promise<void> {
  try {
    const res = await api<Token>("POST", path, { body });
    await setToken(res.token, res.expiresAtUtc);
  } catch (err) {
    if (err instanceof ApiError && err.status === 401) {
      redirect("/login?reason=expired");
    }
    throw err;
  }
}
