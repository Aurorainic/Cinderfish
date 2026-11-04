/**
 * Public surface of the account/session layer.
 *
 * Other workstreams should import from this barrel only:
 *   - `me`, `$i`, `signedIn`, `isSignedIn`, `isAdmin`, `isModerator`
 *   - `login`, `signIn`, `signUp`, `signOut`, `switchAccount`
 *   - `refreshAccount`, `updateProfile`, `restoreSession`
 *   - `authGuard`, `requiresLogin`, `canActivate`, `readRedirectQuery`
 *   - `configureAccountApi`, `AccountApiError`, `classifySigninFailure`
 *   - MiAuth helpers, metadata, formatting helpers and entity types.
 */
export * from "./types";
export * from "./transport";
export * from "./storage";
export * from "./me";
export * from "./session";
export * from "./meta";
export * from "./util";
export * from "./miauth";
export * from "./guard";
