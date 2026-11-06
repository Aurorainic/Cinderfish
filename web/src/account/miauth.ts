/**
 * MiAuth flow, ported from Firefish `pages/miauth.vue`.
 *
 * Cinderfish implements `POST /api/miauth/gen-token` and the polling endpoint
 * `POST /api/miauth/{session}/check` (docs/research/api-contract/auth-users.md).
 * The approval page is `/miauth/:session`. Unlike Misskey the `permission`
 * array is not required and no scope is enforced server-side (only
 * `/streaming` checks `read:account`).
 */
import { accountApi } from "./transport";
import { me } from "./me";
import type { MiAuthTokenRequest } from "./types";

export interface MiAuthRequest {
	session: string;
	name?: string;
	icon?: string;
	/** Comma-separated, as delivered by the `permission` query parameter. */
	permission?: string;
	callback?: string;
}

export interface MiAuthResult {
	token: string;
	/** Fully-qualified callback with `session` appended, or null. */
	callbackUrl: string | null;
}

const FORBIDDEN_CALLBACK_PROTOCOLS = new Set([
	"javascript:",
	"file:",
	"data:",
	"mailto:",
	"tel:",
]);

export function parsePermission(raw: string | undefined): string[] {
	if (raw == null || raw === "") return [];
	return raw
		.split(",")
		.map((entry) => entry.trim())
		.filter((entry) => entry !== "");
}

/** The callback URL Firefish hands back to the requesting app. */
export function buildCallbackUrl(callback: string, session: string): string {
	const url = new URL(callback);
	if (FORBIDDEN_CALLBACK_PROTOCOLS.has(url.protocol)) {
		throw new Error(`Refusing unsafe MiAuth callback protocol: ${url.protocol}`);
	}
	url.searchParams.set("session", session);
	return url.toString();
}

/**
 * Signs a MiAuth session with the current account's token. The caller is
 * responsible for navigating to `callbackUrl` (the page does that after
 * showing the "accepted" state).
 */
export async function acceptMiAuth(
	request: MiAuthRequest,
): Promise<MiAuthResult> {
	const body: MiAuthTokenRequest = {
		session: request.session,
		permission: parsePermission(request.permission),
	};
	if (request.name != null) body.name = request.name;
	if (request.icon != null) body.iconUrl = request.icon;

	const response = await accountApi<{ token: string }>(
		"miauth/gen-token",
		{ ...body },
		me.value?.token ?? null,
	);
	return {
		token: response.token,
		callbackUrl:
			request.callback != null
				? buildCallbackUrl(request.callback, request.session)
				: null,
	};
}

/** Firefish picks an icon from the permission name; kept for parity. */
export function permissionIcon(permission: string): string {
	return permission.includes("write") ? "ph-pencil-simple" : "ph-eye";
}
