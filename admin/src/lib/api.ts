import "server-only";

import { notFound, unstable_rethrow } from "next/navigation";

import { getToken } from "./session";
import type { ErrorBody, FieldError } from "./types";

// Address of the Go API as seen from the Next.js server (e.g. http://api:8080
// inside Docker). It is never sent to the browser.
const API_URL = (process.env.API_URL ?? "http://localhost:8080").replace(/\/+$/, "");

export class ApiError extends Error {
  constructor(
    readonly status: number,
    message: string,
    readonly fields: FieldError[] = [],
    readonly requestId?: string,
  ) {
    super(message);
  }
}

type Method = "GET" | "POST" | "PUT";

type Query = Record<string, string | number | undefined>;

type Options = { body?: unknown; query?: Query; anonymous?: boolean };

/**
 * Calls the API with the token of the signed-in user. Errors become ApiError;
 * a network failure becomes an ApiError with status 0.
 */
export async function api<T>(method: Method, path: string, options: Options = {}): Promise<T> {
  const res = await send(method, path, options);
  if (res.status === 204) {
    return undefined as T;
  }
  return (await res.json()) as T;
}

/** GETs a file from the API; the response body is left unread, to be streamed. */
export async function download(path: string, query?: Query): Promise<Response> {
  return send("GET", path, { query }, "*/*");
}

async function send(method: Method, path: string, options: Options, accept = "application/json"): Promise<Response> {
  const url = new URL(API_URL + path);
  for (const [key, value] of Object.entries(options.query ?? {})) {
    if (value !== undefined && value !== "") {
      url.searchParams.set(key, String(value));
    }
  }

  const headers: Record<string, string> = { Accept: accept };
  if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
  }
  if (!options.anonymous) {
    const token = await getToken();
    if (token) {
      headers.Authorization = `Bearer ${token}`;
    }
  }

  let res: Response;
  try {
    res = await fetch(url, {
      method,
      headers,
      body: options.body === undefined ? undefined : JSON.stringify(options.body),
      cache: "no-store",
    });
  } catch (err) {
    // Next.js signals its own control flow (e.g. a page that must render
    // dynamically) by throwing from fetch: that is not a network failure.
    unstable_rethrow(err);
    console.error(`API unreachable at ${API_URL}`, err);
    throw new ApiError(0, "Não foi possível conectar à API.");
  }

  if (!res.ok) {
    let body: Partial<ErrorBody> = {};
    try {
      body = (await res.json()) as ErrorBody;
    } catch {
      // Not JSON (e.g. a proxy error page).
    }
    throw new ApiError(res.status, body.error ?? res.statusText, body.fields ?? [], body.requestId);
  }
  return res;
}

/** GETs a single record, rendering the not-found page when it does not exist (or belongs to another tenant). */
export async function getOrNotFound<T>(path: string): Promise<T> {
  try {
    return await api<T>("GET", path);
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) {
      notFound();
    }
    throw err;
  }
}
