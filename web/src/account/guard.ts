/**
 * Route guard helpers for the `requiresLogin` meta flag used by Firefish's
 * router (`{ path, component, loginRequired: true }`).
 *
 * The router workstream owns `web/src/router`; it should add
 * `router.beforeEach(authGuard)` and mark protected routes with
 * `meta: { requiresLogin: true }`. Nothing here imports the router instance,
 * so there is no account ↔ router cycle.
 */
import type { RouteLocationNormalized, RouteLocationRaw } from "vue-router";
import { signedIn } from "./me";

declare module "vue-router" {
	interface RouteMeta {
		/** Firefish's `loginRequired`; signed-out visitors are bounced to `/`. */
		requiresLogin?: boolean;
		/** Firefish's `hash: "initialTab"` hint. */
		initialTab?: string;
		/** Force the visitor (welcome) shell even when signed in. */
		visitor?: boolean;
	}
}

export const LOGIN_REDIRECT_QUERY = "redirect";

export function isLoginRequired(to: RouteLocationNormalized): boolean {
	return to.meta.requiresLogin === true;
}

/**
 * `beforeEach` handler. Returns `true` to continue, or a redirect to the
 * timeline/welcome route carrying the original location in `?redirect=`.
 * Firefish's welcome page reads that query and forwards it to `login()`.
 */
export function authGuard(
	to: RouteLocationNormalized,
): true | RouteLocationRaw {
	if (!isLoginRequired(to) || signedIn.value) return true;
	return {
		path: "/",
		query: { [LOGIN_REDIRECT_QUERY]: to.fullPath },
	};
}

/** Reads `?redirect=` from a route location, ignoring unsafe targets. */
export function readRedirectQuery(value: unknown): string | null {
	if (typeof value !== "string" || value === "") return null;
	// Only same-origin absolute paths are accepted; `//evil.example` is not.
	if (!value.startsWith("/") || value.startsWith("//")) return null;
	return value;
}
