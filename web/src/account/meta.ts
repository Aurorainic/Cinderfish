/**
 * Instance metadata used by the account screens.
 *
 * `POST /api/meta` is anonymous and has no error path
 * (docs/research/api-contract/meta-federation.md), so a failure here means the
 * instance is unreachable. The registration fields are read from this object
 * instead of hard-coding Firefish assumptions:
 *  - `disableRegistration` is true for both `invite` and `admin` modes, so the
 *    sign-up form shows the invitation-code field in both (Cinderfish ignores
 *    the code in `admin` mode and answers `REGISTRATION_DISABLED`).
 *  - `features.registration` is `registration_mode == "open"`.
 */
import { accountApi } from "./transport";
import type { MetaLite } from "./types";

let cached: Promise<MetaLite> | null = null;

export function fetchMeta(force = false): Promise<MetaLite> {
	if (!force && cached != null) return cached;
	cached = accountApi<MetaLite>("meta", { detail: false }, null).catch((error) => {
		cached = null;
		throw error;
	});
	return cached;
}

export function resetMetaCache(): void {
	cached = null;
}
