// Helpers shared by the UI atoms, translated from Firefish's filters/user.ts.
import type { UserLite } from "../../api/types";

export interface UserIdentity {
  id?: string;
  username: string;
  host: string | null;
  name?: string | null;
  avatarUrl?: string | null;
}

/** "@alice" on the local instance, "@alice@remote.example" for remote users. */
export function acct(user: Pick<UserLite, "username" | "host">): string {
  return user.host == null ? `@${user.username}` : `@${user.username}@${user.host}`;
}

/** Router path for a user profile. Matches the `/:@acct` route in router/routes.ts. */
export function userPage(user: Pick<UserLite, "username" | "host">): string {
  return `/${acct(user)}`;
}

export function userName(user: Pick<UserLite, "username" | "name">): string {
  return user.name ?? user.username;
}
