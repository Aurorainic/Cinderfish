import { computed, reactive } from "vue";
import { api, ApiError, getToken, setToken } from "../api/client";
import type { MeDetailed } from "../api/types";

// The session is a module-level reactive object: the shell, the router
// guard and every view read the same instance, and a signin anywhere
// updates all of them.
export const session = reactive({
  token: getToken(),
  me: null as MeDetailed | null,
  loading: false,
  ready: false,
});

export const isLoggedIn = computed(() => session.me !== null);
export const isAdmin = computed(() => session.me?.isAdmin === true);

let inflight: Promise<void> | null = null;

// refreshSession loads /api/i once per page load (and after a signin).
// Concurrent callers share one request.
export async function refreshSession(): Promise<void> {
  if (inflight) return inflight;
  if (!getToken()) {
    session.ready = true;
    return;
  }
  inflight = (async () => {
    session.loading = true;
    try {
      session.me = await api.i();
      session.token = getToken();
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) signout();
      session.me = null;
    } finally {
      session.loading = false;
      session.ready = true;
      inflight = null;
    }
  })();
  return inflight;
}

export async function signin(username: string, password: string): Promise<void> {
  const res = await api.signin(username, password);
  setToken(res.i);
  session.token = res.i;
  session.me = await api.i();
  session.ready = true;
}

export async function signup(
  username: string,
  password: string,
  invitationCode: string,
): Promise<void> {
  const res = await api.signup(username, password, invitationCode);
  setToken(res.token);
  session.token = res.token;
  session.me = res;
  session.ready = true;
}

export function signout(): void {
  setToken("");
  session.token = "";
  session.me = null;
}

export interface AccountPatch {
  name?: string | null;
  description?: string | null;
  avatarId?: string | null;
}

// The single write path for the signed-in user's own fields. It replaces the
// stored copy in place so everything projecting from `session.me` (the
// Firefish `me` shim, note headers restamped by the client) sees the new
// values without a reload. Callers: the profile settings view and `me.ts`.
export async function updateAccount(patch: AccountPatch): Promise<void> {
  const updated = await api.iUpdate(patch);
  if (session.me) Object.assign(session.me, updated);
  else session.me = updated;
}
