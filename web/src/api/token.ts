/**
 * Token access shared by the RPC client and the streaming client.
 *
 * Ported pages never read localStorage themselves (runtime adaptation
 * contract); the session layer calls `setToken` on sign-in/sign-out.
 */
const TOKEN_KEY = "cinderfish.token";

let currentToken: string | null = null;
let initialized = false;

function ensureInitialized(): void {
	if (initialized) return;
	initialized = true;
	try {
		currentToken = localStorage.getItem(TOKEN_KEY);
	} catch {
		currentToken = null;
	}
}

export function getToken(): string | null {
	ensureInitialized();
	return currentToken;
}

export function setToken(token: string | null): void {
	ensureInitialized();
	currentToken = token;
	try {
		if (token) localStorage.setItem(TOKEN_KEY, token);
		else localStorage.removeItem(TOKEN_KEY);
	} catch {
		// Storage may be unavailable (private mode); the in-memory token stands.
	}
}

export function hasToken(): boolean {
	return getToken() != null;
}
