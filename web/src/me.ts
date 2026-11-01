// Firefish-shaped projection of the signed-in user.
//
// Firefish keeps `me` as a reactive copy of the account in localStorage and
// every ported component tests it for truthiness (`v-if="me"`, `isSignedIn(me)`)
// and reads its fields directly. Cinderfish has exactly one session, so this
// module projects it instead of storing anything:
//
//   session.me  — the only copy, written by signin/signup/refreshSession
//   me          — a live ES binding refreshed from session.me (null signed out)
//   isAdmin / isModerator — derived booleans with the same live-binding shape
//
// Writing must go through `updateAccount` (or the session store): mutating `me`
// in place would create a second copy that drifts from the session, which is
// the split state the port design forbids.
//
// Reactivity note: the bound value is the session's own reactive object, so
// property reads inside a render are tracked while signed in. A sign-in or
// sign-out replaces the binding, and that replacement is picked up by the
// navigation those flows already perform.
import { watch } from "vue";
import type { MeDetailed } from "./api/types";
import { session } from "./stores/session";

export { updateAccount } from "./stores/session";

let me: MeDetailed | null = null;
export { me };

export let isAdmin = false;
export let isModerator = false;

watch(
  () => session.me,
  (value) => {
    me = value;
    isAdmin = value?.isAdmin === true;
    isModerator = value?.isModerator === true || value?.isAdmin === true;
  },
  { immediate: true, flush: "sync" },
);

/** Firefish's type guard; `me` itself stays a plain binding so `v-if="me"` works. */
export function isSignedIn(value: MeDetailed | null | undefined): value is MeDetailed {
  return value != null;
}
