export type ApiErrorKind = "client" | "server" | "permission";

export interface ApiErrorBody {
	message?: string;
	code?: string;
	id?: string;
	kind?: ApiErrorKind;
	info?: unknown;
}

/** Misskey error envelope: `{ "error": { message, code, id, kind, info? } }`. */
export interface ApiErrorEnvelope {
	error: ApiErrorBody;
}

/** OAuth endpoints use RFC 6749 instead of the Misskey envelope. */
interface OAuthErrorEnvelope {
	error: string;
	error_description?: string;
	state?: string;
}

export class ApiError extends Error {
	/** Machine-readable code (`AUTHENTICATION_FAILED`, `NO_SUCH_KEY`, ...). */
	readonly code: string | null;
	/** Misskey error UUID. Useful for parity debugging, not for branching. */
	readonly errorId: string | null;
	readonly kind: ApiErrorKind | null;
	readonly info: unknown;
	readonly status: number;

	constructor(options: {
		message: string;
		status: number;
		code?: string | null;
		errorId?: string | null;
		kind?: ApiErrorKind | null;
		info?: unknown;
	}) {
		super(options.message);
		this.name = "ApiError";
		this.code = options.code ?? null;
		this.errorId = options.errorId ?? null;
		this.kind = options.kind ?? null;
		this.info = options.info;
		this.status = options.status;
	}

	static network(cause: unknown): ApiError {
		const message =
			cause instanceof Error ? cause.message : "Network request failed";
		return new ApiError({
			message,
			status: 0,
			code: "NETWORK_ERROR",
			kind: "server",
			info: cause,
		});
	}

	/**
	 * Build from a response body. Handles the three shapes the backend emits:
	 * the standard envelope, `signin`'s id-only envelope, and OAuth's
	 * `{ error, error_description }`.
	 */
	static fromResponse(status: number, body: unknown): ApiError {
		if (body != null && typeof body === "object") {
			const record = body as Record<string, unknown>;

			const error = record.error;
			if (error != null && typeof error === "object") {
				const detail = error as ApiErrorBody;
				return new ApiError({
					message: detail.message ?? "Request failed",
					status,
					code: detail.code ?? null,
					errorId: detail.id ?? null,
					kind: detail.kind ?? null,
					info: detail.info,
				});
			}

			if (typeof record.error === "string") {
				const oauth = record as unknown as OAuthErrorEnvelope;
				return new ApiError({
					message: oauth.error_description ?? oauth.error,
					status,
					code: oauth.error,
					kind: "client",
				});
			}
		}

		return new ApiError({
			message: `Request failed with status ${status}`,
			status,
			code: null,
			kind: null,
		});
	}
}

export function isApiError(value: unknown): value is ApiError {
	return value instanceof ApiError;
}

/** Branch on `code`, never on `errorId` (see the contract docs). */
export function hasErrorCode(error: unknown, code: string): boolean {
	return isApiError(error) && error.code === code;
}

/**
 * Well-known `signin` failure ids. The backend returns
 * `{"error":{"id":"..."}}` with no `code` for account/password failures, so
 * these are the only stable discriminators for that endpoint.
 */
export const SIGNIN_ERROR_IDS = {
	noSuchUser: "6cc579cc-885d-43d8-95c2-b8c7fc963280",
	incorrectPassword: "932c904e-9460-45b7-9ce6-7ed33be7eb2c",
	suspended: "e03a5f46-d309-4865-9b69-56282d94e1eb",
} as const;

export function signinFailureReason(error: unknown): keyof typeof SIGNIN_ERROR_IDS | null {
	if (!isApiError(error) || error.errorId == null) return null;
	const entry = Object.entries(SIGNIN_ERROR_IDS).find(
		([, id]) => id === error.errorId,
	);
	return entry == null ? null : (entry[0] as keyof typeof SIGNIN_ERROR_IDS);
}
