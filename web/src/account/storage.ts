/**
 * Browser persistence for the account layer, ported from Firefish
 * `packages/client/src/account.ts`.
 *
 * Firefish stores the signed-in profile under the `account` localStorage key
 * and the multi-account token list in IndexedDB under `accounts`. Cinderfish
 * has no IndexedDB helper yet, so both live in localStorage; the `account`
 * key is kept verbatim so a future Firefish preference import can read it.
 */
import type { MeDetailed, StoredAccount } from "./types";
export type SessionSnapshot = { userId: string; token: string };

export const ACCOUNT_STORAGE_KEY = "account";
export const ACCOUNTS_STORAGE_KEY = "accounts";

function storage(): Storage | null {
	try {
		return typeof localStorage === "undefined" ? null : localStorage;
	} catch {
		return null;
	}
}

function isRecord(value: unknown): value is Record<string, unknown> {
	return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isMeDetailed(value: unknown): value is MeDetailed {
	return (
		isRecord(value) &&
		typeof value.id === "string" &&
		typeof value.username === "string" &&
		typeof value.token === "string" &&
		value.token !== ""
	);
}

/** Reads the persisted profile. Returns null when absent or malformed. */
export function readStoredAccount(): MeDetailed | null {
	const store = storage();
	if (store == null) return null;
	const raw = store.getItem(ACCOUNT_STORAGE_KEY);
	if (raw == null) return null;
	try {
		const parsed: unknown = JSON.parse(raw);
		return isMeDetailed(parsed) ? parsed : null;
	} catch {
		return null;
	}
}

export function writeStoredAccount(account: MeDetailed): void {
	const store = storage();
	if (store == null) return;
	try {
		store.setItem(ACCOUNT_STORAGE_KEY, JSON.stringify(account));
	} catch {
		// Quota/private-mode failures leave the in-memory session usable.
	}
}

export function clearStoredAccount(): void {
	const store = storage();
	if (store == null) return;
	try {
		store.removeItem(ACCOUNT_STORAGE_KEY);
	} catch {
		// Ignore.
	}
}

function isStoredAccount(value: unknown): value is StoredAccount {
	return (
		isRecord(value) &&
		typeof value.id === "string" &&
		typeof value.token === "string" &&
		value.token !== ""
	);
}

export function readStoredAccounts(): StoredAccount[] {
	const store = storage();
	if (store == null) return [];
	const raw = store.getItem(ACCOUNTS_STORAGE_KEY);
	if (raw == null) return [];
	try {
		const parsed: unknown = JSON.parse(raw);
		return Array.isArray(parsed) ? parsed.filter(isStoredAccount) : [];
	} catch {
		return [];
	}
}

function writeStoredAccounts(accounts: StoredAccount[]): void {
	const store = storage();
	if (store == null) return;
	try {
		if (accounts.length === 0) store.removeItem(ACCOUNTS_STORAGE_KEY);
		else store.setItem(ACCOUNTS_STORAGE_KEY, JSON.stringify(accounts));
	} catch {
		// Ignore.
	}
}

/**
 * Adds or replaces the credential for `id`. Firefish ignores duplicates, but
 * replacing is required here because a sign-in can rotate the native token.
 */
export function addStoredAccount(account: StoredAccount): void {
	const accounts = readStoredAccounts();
	const index = accounts.findIndex((entry) => entry.id === account.id);
	if (index === -1) accounts.push(account);
	else accounts[index] = account;
	writeStoredAccounts(accounts);
}

export function removeStoredAccount(id: string): void {
	writeStoredAccounts(readStoredAccounts().filter((entry) => entry.id !== id));
}

export function findStoredAccount(id: string): StoredAccount | null {
	return readStoredAccounts().find((entry) => entry.id === id) ?? null;
}

export function toSessionSnapshot(account: MeDetailed): SessionSnapshot | null {
	if (account.token == null || account.token === "") return null;
	return { userId: account.id, token: account.token };
}
