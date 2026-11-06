/**
 * Account-layer entity types.
 *
 * The shared API contract types live in `web/src/api/types.ts`; this module
 * re-exports the ones the account layer needs and adds only the client-side
 * shapes (the attached token, the multi-account credential entry, and the
 * editable profile subset).
 *
 * Contract source: docs/research/api-contract/auth-users.md.
 */
import type { MeDetailed as ApiMeDetailed, ProfileField } from "../api/types";

export type {
	DriveFile,
	MetaLite,
	ProfileField,
	SigninResponse,
	SigninStep,
	SignupResponse,
	TwoFactorRegistration,
	UserDetailed,
	UserLite,
} from "../api/types";

/** Canonical name in `web/src/api/types.ts`. */
export type { UserNameAvailability as UsernameAvailableResponse } from "../api/types";

/**
 * `MeDetailed` plus the raw token.
 *
 * `POST /api/i` never returns the token, the account layer stores it next to
 * the profile. `email`, `emailVerified` and `securityKeysList` are only
 * present when the request used a 16-character native token.
 */
export interface MeDetailed extends ApiMeDetailed {
	/** Browser-only; never sent back to the server. */
	token?: string;
}

/** A signed-in profile with its token guaranteed present. */
export type Account = MeDetailed & { token: string };

export type UserVisibility = "public" | "followers" | "private";

/** `POST /api/i/update` — fields omitted are left untouched. */
export interface ProfileUpdateRequest {
	name?: string | null;
	description?: string | null;
	avatarId?: string | null;
	followersVisibility?: UserVisibility | null;
	followingVisibility?: UserVisibility | null;
	location?: string | null;
	birthday?: string | null;
	lang?: string | null;
	fields?: ProfileField[];
	noCrawle?: boolean;
	preventAiLearning?: boolean;
	isExplorable?: boolean;
	hideOnlineStatus?: boolean;
	publicReactions?: boolean;
	alwaysMarkNsfw?: boolean;
	autoSensitive?: boolean;
	carefulBot?: boolean;
	autoAcceptFollowed?: boolean;
	injectFeaturedNote?: boolean;
}

/** Stored credential entry for the account switcher. */
export interface StoredAccount {
	id: string;
	token: string;
}

/** MiAuth token request, `POST /api/miauth/gen-token`. */
export interface MiAuthTokenRequest {
	session: string;
	name?: string;
	description?: string;
	iconUrl?: string;
	permission: string[];
}
