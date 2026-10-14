import "server-only";

import { cookies } from "next/headers";

// The API token lives only in this httpOnly cookie: the browser never sees it
// and every API call is made by the Next.js server.
const COOKIE_NAME = "ept_session";

// Secure cookies are only sent over HTTPS (browsers also accept them on
// http://localhost). Set COOKIE_SECURE=false to serve the panel over plain
// HTTP on another host.
function secure(): boolean {
  const raw = process.env.COOKIE_SECURE;
  if (raw) {
    return raw.toLowerCase() === "true";
  }
  return process.env.NODE_ENV === "production";
}

export async function getToken(): Promise<string | undefined> {
  return (await cookies()).get(COOKIE_NAME)?.value;
}

/** Only callable from Server Actions and Route Handlers. */
export async function setToken(token: string, expiresAtUtc: string): Promise<void> {
  (await cookies()).set(COOKIE_NAME, token, {
    httpOnly: true,
    secure: secure(),
    sameSite: "lax",
    path: "/",
    expires: new Date(expiresAtUtc),
  });
}

/** Only callable from Server Actions and Route Handlers. */
export async function clearToken(): Promise<void> {
  (await cookies()).delete(COOKIE_NAME);
}
