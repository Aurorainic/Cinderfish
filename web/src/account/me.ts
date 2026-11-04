/**
 * Signed-in account store, ported from Firefish's `$i` object in
 * `account.ts` plus the account bucket of `store.ts`.
 *
 * Persistence is delegated to `storage.ts` (Firefish's `account` key), and
 * the store layer is notified through `store-composables/session-bridge.ts`.
 * The raw token is also mirrored to `cinderfish.token`, the key the RPC and
 * streaming clients read when they are used without an explicit token.
 */
import { computed, ref, type ComputedRef, type Ref } from "vue";
import {
	ACCOUNT_STORAGE_KEY,
	clearStoredAccount,
	readStoredAccount,
	writeStoredAccount,
} from "./storage";
import type { MeDetailed } from "./types";

/** Token key owned by the RPC/streaming clients (`web/src/api/token.ts`). */
export const CLIENT_TOKEN_KEY = "cinderfish.token";

function mirrorToken(account: MeDetailed | null): void {
	try {
		if (account?.token != null && account.token !== "") {
			localStorage.setItem(CLIENT_TOKEN_KEY, account.token);
		} else {
			localStorage.removeItem(CLIENT_TOKEN_KEY);
		}
	} catch {
		// Private mode / quota: the in-memory session still works.
	}
}

function publish(account: MeDetailed | null): void {
	mirrorToken(account);
}

/** Firefish's `$i`. Hydrated from localStorage at module load. */
export const me: Ref<MeDetailed | null> = ref(readStoredAccount());

/** Reactive "am I signed in" for layouts and guards. */
export const signedIn: ComputedRef<boolean> = computed(() => me.value != null);

/** Firefish's `$i` spelling, for future page ports that expect it. */
export { me as $i };

// Make the restored session visible to stores created at module load.
publish(me.value);

/** Replaces the profile wholesale (`i`, `i/update`, `signup` all return one). */
export function setMe(account: MeDetailed | null): void {
	if (account == null) {
		clearStoredAccount();
	} else {
		writeStoredAccount(account);
	}
	me.value = account;
	publish(account);
}

/** Field-level merge, used where an endpoint does not fill every field. */
export function updateMe(patch: Partial<MeDetailed>): void {
	if (me.value == null) return;
	setMe({ ...me.value, ...patch });
}

export function isSignedIn(account: MeDetailed | null): account is MeDetailed {
	return account != null;
}

export const isModerator = (): boolean =>
	me.value != null && (me.value.isModerator || me.value.isAdmin);

export const isAdmin = (): boolean => me.value?.isAdmin === true;

/** Cinderfish has no separate emoji-moderator role. */
export const isEmojiMod = (): boolean => isModerator();

export function accountAcct(account: MeDetailed): string {
	return account.host != null && account.host !== ""
		? `@${account.username}@${account.host}`
		: `@${account.username}`;
}

// Another tab signed in, signed out or switched accounts: mirror it here
// without writing the key back (which would echo between tabs).
if (typeof window !== "undefined") {
	window.addEventListener("storage", (event) => {
		if (event.key !== ACCOUNT_STORAGE_KEY) return;
		const next = readStoredAccount();
		if (next?.id === me.value?.id && next?.token === me.value?.token) return;
		me.value = next;
		publish(next);
	});
}
