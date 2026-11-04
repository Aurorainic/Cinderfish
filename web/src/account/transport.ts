/**
 * Transport seam for the account layer.
 *
 * The account layer owns its own minimal RPC call so it can boot (and be
 * tested) before `web/src/api` finishes loading; the integration layer may
 * replace it with the shared client through `configureAccountApi()`.
 *
 * Contract notes (docs/research/api-contract/auth-users.md, 通用约定):
 *  - every RPC is `POST /api/<endpoint>` with a JSON body;
 *  - the token goes in the body as `i` and takes precedence over
 *    `Authorization: Bearer`;
 *  - success bodies are bare values (no `{result}` wrapper);
 *  - errors use `{ error: { message, code, id, kind, info? } }`, except
 *    `signin`, which uses Misskey's id-only envelope `{ error: { id } }`.
 */
export type ApiCall = <T>(
	endpoint: string,
	params?: Record<string, unknown>,
	token?: string | null,
) => Promise<T>;

export interface ApiErrorInfo {
	message: string;
	code?: string;
	id?: string;
	kind?: "client" | "server" | "permission";
	info?: unknown;
}

export class AccountApiError extends Error implements ApiErrorInfo {
	readonly code?: string;
	readonly id?: string;
	readonly kind?: "client" | "server" | "permission";
	readonly info?: unknown;
	readonly status: number;

	constructor(info: ApiErrorInfo, status: number) {
		super(info.message);
		this.name = "AccountApiError";
		this.code = info.code;
		this.id = info.id;
		this.kind = info.kind;
		this.info = info.info;
		this.status = status;
	}
}

/** Error ids Firefish's `MkSignin.vue` branches on; Cinderfish reuses them. */
export const SIGNIN_ERROR_IDS = {
	noSuchUser: "6cc579cc-885d-43d8-95c2-b8c7fc963280",
	incorrectPassword: "932c904e-9460-45b7-9ce6-7ed33be7eb2c",
	suspended: "e03a5f46-d309-4865-9b69-56282d94e1eb",
	rateLimited: "22d05606-fbcf-421a-a2db-b32610dcfd1b",
} as const;

export type SigninFailure =
	| "no-such-user"
	| "incorrect-password"
	| "suspended"
	| "rate-limited"
	| "unknown";

export function classifySigninFailure(error: unknown): SigninFailure {
	if (!(error instanceof AccountApiError)) return "unknown";
	switch (error.id) {
		case SIGNIN_ERROR_IDS.noSuchUser:
			return "no-such-user";
		case SIGNIN_ERROR_IDS.incorrectPassword:
			return "incorrect-password";
		case SIGNIN_ERROR_IDS.suspended:
			return "suspended";
		case SIGNIN_ERROR_IDS.rateLimited:
			return "rate-limited";
		default:
			return "unknown";
	}
}

function isRecord(value: unknown): value is Record<string, unknown> {
	return typeof value === "object" && value !== null && !Array.isArray(value);
}

function parseErrorBody(body: unknown, status: number): AccountApiError {
	if (isRecord(body) && isRecord(body.error)) {
		const envelope = body.error;
		const message =
			typeof envelope.message === "string"
				? envelope.message
				: typeof envelope.code === "string"
					? envelope.code
					: "Authentication failed.";
		return new AccountApiError(
			{
				message,
				code: typeof envelope.code === "string" ? envelope.code : undefined,
				id: typeof envelope.id === "string" ? envelope.id : undefined,
				kind:
					envelope.kind === "client" ||
					envelope.kind === "server" ||
					envelope.kind === "permission"
						? envelope.kind
						: undefined,
				info: envelope.info,
			},
			status,
		);
	}
	return new AccountApiError(
		{ message: `Request failed with status ${status}.` },
		status,
	);
}

/**
 * Same-origin RPC. Cinderfish serves the API from the same origin as the SPA
 * (`/api/*`), and the Vite dev server proxies it, so no base URL is needed.
 */
const fallbackApi: ApiCall = async <T>(
	endpoint: string,
	params: Record<string, unknown> = {},
	token?: string | null,
): Promise<T> => {
	// `i` must be the first body field; spreading params afterwards lets an
	// explicit `i` in params win without moving the key.
	const body: Record<string, unknown> =
		token != null && token !== "" ? { i: token, ...params } : { ...params };

	const response = await fetch(`/api/${endpoint}`, {
		method: "POST",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(body),
		credentials: "omit",
		cache: "no-cache",
	});

	if (response.status === 204) return undefined as T;

	const text = await response.text();
	let parsed: unknown = null;
	if (text !== "") {
		try {
			parsed = JSON.parse(text);
		} catch {
			parsed = null;
		}
	}

	if (!response.ok) throw parseErrorBody(parsed, response.status);
	return parsed as T;
};

let configuredApi: ApiCall | null = null;

/**
 * Installs the shared RPC client (`web/src/api`) or a test double.
 * `configureAccountApi()` restores the built-in call.
 */
export function configureAccountApi(next: ApiCall | null = null): void {
	configuredApi = next;
}

export const accountApi: ApiCall = <T>(
	endpoint: string,
	params?: Record<string, unknown>,
	token?: string | null,
): Promise<T> => (configuredApi ?? fallbackApi)<T>(endpoint, params, token);
