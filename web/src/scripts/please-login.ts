// Ported from firefish@b425f61:packages/client/src/scripts/please-login.ts (AGPL-3.0-only)
//
// Firefish opens MkSigninDialog and throws when no follow-up path is given, so
// the caller aborts. Cinderfish has no sign-in dialog yet: the shim sends the
// visitor to "/" (the welcome route, which carries the sign-in form) and shows
// a toast. The throw is kept because ported handlers rely on it to stop.
import { me } from "../account";
import { t } from "../i18n";
import { toast } from "../os";
import { router } from "../router";

export function pleaseLogin(path?: string): void {
	if (me.value != null) return;

	const target = path ?? "/";
	if (router.currentRoute.value.path !== target) {
		void router.push(target);
	}
	toast(t("signinRequired"), "info");

	if (path == null) throw new Error("Sign-in required.");
}
