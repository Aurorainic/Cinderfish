// RPC layer for the Cinderfish HTTP API.
//
// Contract facts this file is built around (docs/research/api-contract/):
//   * every /api route is POST + JSON (GET only for ping and the federation
//     allowlist), the token travels in the body as `i`;
//   * errors use `{ error: { message, code, id, kind, info? } }`, with the
//     documented exceptions handled by ApiError.fromResponse();
//   * there is no CORS layer, so requests are same-origin only;
//   * list endpoints return a bare array.
//
// `api()` is typed against the EndpointMap in ./endpoints; pass an endpoint
// string that is not in the map and the loose overload returns `unknown`.
import { ref } from "vue";
import type { EndpointName, EndpointParams, EndpointResult } from "./endpoints";
import { ApiError } from "./errors";
import { getToken as readSharedToken } from "./token";

export { ApiError } from "./errors";
export type { ApiErrorBody, ApiErrorKind } from "./errors";

/** Requests in flight; a global loading indicator can watch this. */
export const pendingApiRequestsCount = ref(0);

/**
 * Endpoints that are registered as GET instead of POST.
 * Everything else uses POST, including endpoints whose name looks like a read.
 */
export const GET_ENDPOINTS: ReadonlySet<string> = new Set([
  "ping",
  "admin/federation/allowlist",
  "admin/federation/mode",
]);

let explicitToken: string | null = null;

/** Called by the account layer on sign-in / sign-out. */
export function setAuthToken(token: string | null): void {
  explicitToken = token;
}

/** Explicit token first, then the shared session token (api/token.ts). */
export function getAuthToken(): string | null {
  return explicitToken ?? readSharedToken();
}

function apiBaseUrl(): string {
  if (typeof location === "undefined") return "/api";
  return `${location.origin}/api`;
}

function resolveUrl(endpoint: string): string {
  if (endpoint.includes("://")) return endpoint;
  return `${apiBaseUrl()}/${endpoint.replace(/^\/+/, "")}`;
}

async function parseBody(response: Response): Promise<unknown> {
  const text = await response.text();
  if (text.length === 0) return null;
  try {
    return JSON.parse(text) as unknown;
  } catch {
    return text;
  }
}

async function send<T>(endpoint: string, init: RequestInit): Promise<T> {
  pendingApiRequestsCount.value += 1;
  try {
    let response: Response;
    try {
      response = await fetch(resolveUrl(endpoint), {
        credentials: "omit",
        cache: "no-cache",
        ...init,
      });
    } catch (cause) {
      throw ApiError.network(cause);
    }

    const body = await parseBody(response);

    if (!response.ok) {
      throw ApiError.fromResponse(response.status, body);
    }

    // A 200 carrying an error envelope still means failure.
    if (body != null && typeof body === "object" && "error" in body) {
      throw ApiError.fromResponse(response.status, body);
    }

    return body as T;
  } finally {
    pendingApiRequestsCount.value -= 1;
  }
}

export function api<E extends EndpointName>(
  endpoint: E,
  params?: EndpointParams<E>,
  token?: string | null,
): Promise<EndpointResult<E>>;
export function api<T = unknown>(
  endpoint: string,
  params?: Record<string, unknown>,
  token?: string | null,
): Promise<T>;
export function api(
  endpoint: string,
  params: Record<string, unknown> = {},
  token?: string | null,
): Promise<unknown> {
  const authToken = token ?? getAuthToken();
  const body: Record<string, unknown> = { ...params };
  if (authToken != null) body.i = authToken;

  return send(endpoint, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

/** GET variant for the two documented GET routes. */
export function apiGet<E extends EndpointName>(
  endpoint: E,
  params?: EndpointParams<E>,
  token?: string | null,
): Promise<EndpointResult<E>>;
export function apiGet<T = unknown>(
  endpoint: string,
  params?: Record<string, unknown>,
  token?: string | null,
): Promise<T>;
export function apiGet(
  endpoint: string,
  params: Record<string, unknown> = {},
  token?: string | null,
): Promise<unknown> {
  const authToken = token ?? getAuthToken();
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value == null) continue;
    query.set(key, String(value));
  }
  const suffix = query.toString();
  const headers: Record<string, string> = {};
  if (authToken != null) headers.Authorization = `Bearer ${authToken}`;

  return send(suffix.length > 0 ? `${endpoint}?${suffix}` : endpoint, {
    method: "GET",
    headers,
  });
}

/**
 * Multipart upload. Unlike JSON RPC, drive/files/create only accepts the token
 * in the Authorization header (contract: drive-media.md).
 */
export function apiForm<T = unknown>(
  endpoint: string,
  form: FormData,
  token?: string | null,
): Promise<T> {
  const authToken = token ?? getAuthToken();
  const headers: Record<string, string> = {};
  if (authToken != null) headers.Authorization = `Bearer ${authToken}`;

  return send<T>(endpoint, { method: "POST", headers, body: form });
}

export interface DownloadedFile {
  blob: Blob;
  filename: string | null;
}

/**
 * POST an RPC that answers with a file body instead of JSON (currently only
 * `i/export`). Errors still use the JSON error envelope, so a failed request is
 * raised as ApiError exactly like `api()`.
 */
export async function apiDownload(
  endpoint: string,
  params: Record<string, unknown> = {},
  token?: string | null,
): Promise<DownloadedFile> {
  const authToken = token ?? getAuthToken();
  const body: Record<string, unknown> = { ...params };
  if (authToken != null) body.i = authToken;

  pendingApiRequestsCount.value += 1;
  try {
    let response: Response;
    try {
      response = await fetch(resolveUrl(endpoint), {
        method: "POST",
        credentials: "omit",
        cache: "no-cache",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
    } catch (cause) {
      throw ApiError.network(cause);
    }

    if (!response.ok) {
      const text = await response.text();
      let parsed: unknown = null;
      try {
        parsed = JSON.parse(text) as unknown;
      } catch {
        parsed = text;
      }
      throw ApiError.fromResponse(response.status, parsed);
    }

    const disposition = response.headers.get("content-disposition") ?? "";
    const match = disposition.match(/filename="?([^";]+)"?/);
    return {
      blob: await response.blob(),
      filename: match?.[1] ?? null,
    };
  } finally {
    pendingApiRequestsCount.value -= 1;
  }
}

/** Picks GET or POST from the endpoint table. */
export function call<E extends EndpointName>(
  endpoint: E,
  params?: EndpointParams<E>,
  token?: string | null,
): Promise<EndpointResult<E>> {
  if (GET_ENDPOINTS.has(endpoint)) {
    return apiGet<EndpointResult<E>>(
      endpoint,
      params as Record<string, unknown>,
      token,
    );
  }
  return api<EndpointResult<E>>(
    endpoint,
    params as Record<string, unknown>,
    token,
  );
}

/** Media URLs are root-relative locally and absolute on S3/public storage. */
export function resolveMediaUrl(url: string | null | undefined): string {
  if (url == null || url === "") return "";
  if (/^(https?:|data:|blob:)/.test(url)) return url;
  if (url.startsWith("/")) {
    return typeof location === "undefined" ? url : `${location.origin}${url}`;
  }
  return url;
}

/** Convenience wrappers used during bootstrap. */
export function fetchMeta(detail = true): Promise<unknown> {
  return api("meta", { detail });
}

export function fetchMe(): Promise<unknown> {
  return api("i", {});
}
