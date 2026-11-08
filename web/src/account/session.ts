/**
 * Account lifecycle, ported from Firefish `account.ts`.
 *
 * Firefish reloads the whole document after login/logout. The port resolves
 * inside the SPA and exposes lifecycle hooks so the store and streaming
 * layers can hydrate without this module importing them (that would create a
 * cycle: stores read the session at module load).
 *
 * Token persistence and the reactive profile live in `me.ts` / `storage.ts`;
 * this module is the lifecycle API.
 */
import type { RouteLocationNormalized } from "vue-router";
import { computed, type ComputedRef } from "vue";
import {
	addStoredAccount,
	findStoredAccount,
	removeStoredAccount,
} from "./storage";
import { AccountApiError, accountApi } from "./transport";
import { me, setMe, updateMe } from "./me";
import type {
	MeDetailed,
	ProfileUpdateRequest,
	SigninStep,
	SignupResponse,
} from "./types";

export interface SignInResult {
	account: MeDetailed;
	token: string;
}

/**
 * The password was accepted but the account has TOTP enabled: the caller
 * must repeat `signIn` with the six-digit code from the authenticator app.
 */
export interface TwoFactorRequired {
	twoFactorRequired: true;
}

export type SignInOutcome = SignInResult | TwoFactorRequired;

/**
 * Hooks installed by the app shell (`main.ts`). Kept optional so the account
 * layer can be used on its own in tests and in the sign-in dialog.
 */
export interface AccountLifecycleHooks {
	onSignedIn?: (account: MeDetailed, token: string) => void | Promise<void>;
	onSignedOut?: () => void | Promise<void>;
}

let hooks: AccountLifecycleHooks = {};

export function setAccountLifecycleHooks(next: AccountLifecycleHooks): void {
	hooks = next;
}

function isAuthenticationFailure(error: unknown): boolean {
	return (
		error instanceof AccountApiError &&
		(error.status === 401 || error.code === "AUTHENTICATION_FAILED")
	);
}

/**
 * `POST /api/i` with an explicit token. The response never carries the token,
 * so it is attached to the returned object (Firefish does the same).
 */
export async function fetchMe(token: string): Promise<MeDetailed> {
	const account = await accountApi<MeDetailed>("i", {}, token);
	return { ...account, token };
}

export interface LoginOptions {
	/** Navigate here after the profile has been stored. */
	redirect?: string;
}

function commit(account: MeDetailed): MeDetailed {
	setMe(account);
	if (account.token != null && account.token !== "") {
		addStoredAccount({ id: account.id, token: account.token });
	}
	return account;
}

async function notifySignedIn(account: MeDetailed, token: string): Promise<void> {
	try {
		await hooks.onSignedIn?.(account, token);
	} catch (error) {
		console.warn("[account] onSignedIn hook failed", error);
	}
}

/**
 * Signs in with an already-obtained token (Firefish's `login`). Useful for
 * account switching and for `?token=` hand-offs.
 */
export async function login(
	token: string,
	options: LoginOptions = {},
): Promise<MeDetailed> {
	const account = await fetchMe(token);
	commit(account);
	await notifySignedIn(account, token);
	if (options.redirect != null) window.location.assign(options.redirect);
	return account;
}

/**
 * Password sign-in (`POST /api/signin`). When the account has TOTP enabled
 * and no `token` is supplied, the server answers an empty body and this
 * function reports `twoFactorRequired`; the caller repeats the call with the
 * six-digit code, which returns the normal `{ finished, id, i }` body.
 */
export async function signIn(
	username: string,
	password: string,
	token?: string,
	options: LoginOptions = {},
): Promise<SignInOutcome> {
	const params: { username: string; password: string; token?: string } = {
		username,
		password,
	};
	if (token != null && token !== "") params.token = token;
	const response = await accountApi<SigninStep>("signin", params, null);
	if (response.finished !== true) {
		return { twoFactorRequired: true };
	}
	const account = await login(response.i, options);
	return { account, token: response.i };
}

/** Firefish's spelling in some locales; kept as an alias. */
export const signin = signIn;

export interface SignUpOptions extends LoginOptions {
	invitationCode?: string;
	/** Captcha tokens, one of them depending on the instance's provider. */
	captcha?: {
		hcaptcha?: string;
		recaptcha?: string;
		turnstile?: string;
	};
}

/**
 * `POST /api/signup` returns MeDetailed plus the 16-character native token,
 * so the account is usable without the extra `signin` round trip Firefish
 * performs. `POST /api/i` still follows, because signup's response leaves
 * unread counts and pins at their defaults.
 */
export async function signUp(
	username: string,
	password: string,
	invitationCodeOrOptions?: string | SignUpOptions,
): Promise<SignInResult> {
	const options: SignUpOptions =
		typeof invitationCodeOrOptions === "string"
			? { invitationCode: invitationCodeOrOptions }
			: (invitationCodeOrOptions ?? {});

	const response = await accountApi<SignupResponse>(
		"signup",
		{
			username,
			password,
			...(options.invitationCode != null && options.invitationCode !== ""
				? { invitationCode: options.invitationCode }
				: {}),
			// Misskey's signup body uses kebab-case captcha fields.
			...(options.captcha?.hcaptcha != null && options.captcha.hcaptcha !== ""
				? { "hcaptcha-response": options.captcha.hcaptcha }
				: {}),
			...(options.captcha?.recaptcha != null && options.captcha.recaptcha !== ""
				? { "g-recaptcha-response": options.captcha.recaptcha }
				: {}),
			...(options.captcha?.turnstile != null && options.captcha.turnstile !== ""
				? { "turnstile-response": options.captcha.turnstile }
				: {}),
		},
		null,
	);

	const account = await fetchMe(response.token);
	commit(account);
	await notifySignedIn(account, response.token);
	if (options.redirect != null) window.location.assign(options.redirect);
	return { account, token: response.token };
}

/** Firefish's spelling in some locales; kept as an alias. */
export const signup = signUp;

/** `POST /api/username/available`. */
export async function checkUsername(username: string): Promise<boolean> {
	const result = await accountApi<{ available: boolean }>(
		"username/available",
		{ username },
		null,
	);
	return result.available;
}

/** Alias matching the canonical API helper name. */
export const checkUsernameAvailable = checkUsername;

/**
 * MiAuth token exchange. Cinderfish implements only `miauth/gen-token`
 * (docs/research/api-contract/auth-users.md); the approval page is ours.
 */
export async function miauthGenToken(options: {
	session: string;
	name?: string;
	description?: string;
	iconUrl?: string;
	permission?: string[];
}): Promise<string> {
	const result = await accountApi<{ token: string }>(
		"miauth/gen-token",
		{ ...options },
		me.value?.token ?? null,
	);
	return result.token;
}

/** Alias used by the MiAuth page port. */
export const genMiAuthToken = miauthGenToken;

/**
 * Refreshes the cached profile. A rejected token signs the account out;
 * network failures keep the cached profile so offline reloads still render.
 */
export async function refreshAccount(): Promise<MeDetailed | null> {
	const current = me.value;
	if (current?.token == null) return null;
	try {
		const fresh = await fetchMe(current.token);
		setMe(fresh);
		return fresh;
	} catch (error) {
		if (isAuthenticationFailure(error)) await signOut();
		return null;
	}
}

/** Composable-friendly alias. */
export const refreshMe = refreshAccount;

/**
 * Boot hook: restores the session from the persisted token and refreshes the
 * profile. Safe to call more than once.
 */
export async function restoreSession(): Promise<MeDetailed | null> {
	if (me.value?.token == null) {
		// No cached profile: a token can still exist if storage was partially
		// cleared. Try it once so a stale credential is not silently kept.
		const token = readPersistedClientToken();
		if (token == null) return null;
		try {
			return await login(token);
		} catch {
			clearPersistedClientToken();
			return null;
		}
	}
	return refreshAccount();
}

function readPersistedClientToken(): string | null {
	try {
		const token = localStorage.getItem("cinderfish.token");
		return token == null || token === "" ? null : token;
	} catch {
		return null;
	}
}

function clearPersistedClientToken(): void {
	try {
		localStorage.removeItem("cinderfish.token");
	} catch {
		// Ignore.
	}
}

/** Aliases used by the app boot sequence in `main.ts`. */
export const initAccount = restoreSession;
export const reloadAccountFromStorage = initAccount;

/** Clears the current account. Other stored accounts stay for the switcher. */
export async function signOut(): Promise<void> {
	const current = me.value;
	if (current != null) removeStoredAccount(current.id);
	setMe(null);
	try {
		await hooks.onSignedOut?.();
	} catch (error) {
		console.warn("[account] onSignedOut hook failed", error);
	}
}

/** Aliases matching Firefish's `signout` / settings wording. */
export const signout = signOut;
export const logout = signOut;

/** Switches to another stored account by re-fetching `POST /api/i`. */
export async function switchAccount(id: string): Promise<MeDetailed> {
	const stored = findStoredAccount(id);
	if (stored == null) throw new Error(`No stored credential for account ${id}`);
	return login(stored.token);
}

/**
 * `POST /api/i/update`.
 *
 * The response fills counts and unread flags but always reports
 * `pinnedNoteIds` / `pinnedNotes` as `[]` (i_update.go does not call
 * `applyPinnedNotes`), so the existing pin state is carried over instead of
 * being wiped. Pins only change through `i/pin` / `i/unpin`.
 */
export async function updateProfile(
	patch: ProfileUpdateRequest,
): Promise<MeDetailed> {
	const current = me.value;
	if (current?.token == null) {
		throw new Error("updateProfile() called without a signed-in account");
	}
	const fresh = await accountApi<MeDetailed>(
		"i/update",
		{ ...patch },
		current.token,
	);
	const merged: MeDetailed = {
		...current,
		...fresh,
		token: current.token,
		pinnedNoteIds: current.pinnedNoteIds,
		pinnedNotes: current.pinnedNotes,
	};
	setMe(merged);
	return merged;
}

/** Field-level local update (Firefish's `updateAccount`). */
export function updateAccount(patch: Partial<MeDetailed>): void {
	updateMe(patch);
}

/** Route guard helper: `meta.requiresLogin` marks guarded routes. */
export function requiresLogin(route: RouteLocationNormalized): boolean {
	return route.meta.requiresLogin === true;
}

/** `beforeEach` predicate form expected by the router workstream. */
export function canActivate(route: RouteLocationNormalized): boolean {
	if (!requiresLogin(route)) return true;
	return me.value != null;
}

/** Reactive view for components that prefer a composable. */
export function useMe(): {
	me: typeof me;
	signedIn: ComputedRef<boolean>;
} {
	return { me, signedIn: computed(() => me.value != null) };
}

/**
 * Firefish-style composable used by the account chip and settings pages.
 * Kept in addition to `useMe()` so either naming works.
 */
export function useAccount(): {
	me: typeof me;
	isSignedIn: ComputedRef<boolean>;
	signIn: typeof signIn;
	signUp: typeof signUp;
	signOut: typeof signOut;
	refreshAccount: typeof refreshAccount;
	updateMe: typeof updateProfile;
} {
	return {
		me,
		isSignedIn: computed(() => me.value != null),
		signIn,
		signUp,
		signOut,
		refreshAccount,
		updateMe: updateProfile,
	};
}
