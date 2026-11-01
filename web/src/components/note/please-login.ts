// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/scripts/please-login.ts
// (adapted: Firefish pops a sign-in dialog from MkSigninDialog; Cinderfish
// has no sign-in dialog component, so the port redirects to the login page
// and the caller sees `false` instead of an exception).
import { isSignedIn, me } from "../../me";
import { alert } from "../../os";

// TODO: replace the redirect with a sign-in popup once a MkSigninDialog
// equivalent exists; i18n lacks a `signinRequired` key, so the message is
// hardcoded Chinese for now.
export function pleaseLogin(path?: string): boolean {
	if (isSignedIn(me)) return true;
	if (path) {
		window.location.href = path;
	} else {
		alert({ type: "info", text: "请先登录。" });
	}
	return false;
}
