// Ported from firefish@b425f61:packages/client/src/instance.ts (AGPL-3.0-only)
//
// Reactive instance metadata, filled from POST /api/meta {detail:true}
// (docs/research/api-contract/meta-federation.md). The backend returns
// Misskey-shaped meta: there is no "host"/"softwareName"/"defaultReaction" and
// the timeline switches live under "features" instead of Firefish's
// disable*Timeline booleans. Both spellings are provided so ported Firefish
// components keep reading what they expect, while new code can use the
// backend's own fields.
import { computed, reactive, type ComputedRef } from "vue";
import { api } from "./api/client";
import type { EmojiSimple, MetaDetailed } from "./api/types";
import { applyInstanceInfo, host } from "./config";

export interface InstanceMeta extends MetaDetailed {
	/** Synthesised: the instance's own host, matching Firefish's instance.host. */
	host: string;
	/** Synthesised: this client's software name/version. */
	softwareName: string;
	softwareVersion: string;
	/** Synthesised: reaction sent when the user clicks the plain "star". */
	defaultReaction: string;
	/** Synthesised from features.localTimeline (Firefish naming). */
	disableLocalTimeline: boolean;
	/** Synthesised from features.globalTimeline (Firefish naming). */
	disableGlobalTimeline: boolean;
	/** Always true: the backend has no recommended-timeline endpoint. */
	disableRecommendedTimeline: boolean;
	/** Synthesised from iconUrl; Firefish components fall back to it. */
	faviconUrl: string;
	/** Custom emoji registry; bare meta does not include it (see api("emojis")). */
	emojis: EmojiSimple[];
	/** Synthesised: backend caption limit, Firefish default when unset. */
	maxCaptionTextLength: number;
	/** Synthesised: the backend has no donation link. */
	donationLink: string | null;
}

const CACHE_KEY = "instance";

/**
 * Optimistic feature defaults: the timelines are assumed available until meta
 * says otherwise, so the timeline tabs do not flicker while the request is in
 * flight. Everything else is conservative.
 */
function defaultFeatures(): InstanceMeta["features"] {
	return {
		localTimeline: true,
		globalTimeline: true,
		registration: false,
		emailRequiredForSignup: false,
		hcaptcha: false,
		recaptcha: false,
		turnstile: false,
		objectStorage: false,
		serviceWorker: false,
		miauth: false,
	};
}

/** Backend meta has no emoji/ads payload; keep the shapes components expect. */
function defaults(): InstanceMeta {
	return {
		host,
		uri: host,
		name: "",
		version: "",
		softwareName: "cinderfish",
		softwareVersion: "",
		defaultReaction: "⭐",
		faviconUrl: "",
		disableLocalTimeline: false,
		disableGlobalTimeline: false,
		disableRecommendedTimeline: true,
		emojis: [],
		ads: [],
		maxCaptionTextLength: 512,
		donationLink: null,
		features: defaultFeatures(),
		// MetaLite/MetaDetailed fields the type requires but that only arrive
		// with the first meta response.
	} as unknown as InstanceMeta;
}

function readCache(): Record<string, unknown> | null {
	try {
		const raw = localStorage.getItem(CACHE_KEY);
		if (raw == null) return null;
		const parsed = JSON.parse(raw) as unknown;
		return parsed != null && typeof parsed === "object"
			? (parsed as Record<string, unknown>)
			: null;
	} catch {
		return null;
	}
}

export const instance: InstanceMeta = reactive({
	...defaults(),
	...readCache(),
}) as InstanceMeta;

/** Re-derives the Firefish-named aliases from the backend-shaped fields. */
function applyAliases(): void {
	// A cached object from an older build, or an error response without
	// features, must not stop the aliases from being computed.
	if (instance.features == null) instance.features = defaultFeatures();
	instance.disableLocalTimeline = !instance.features.localTimeline;
	instance.disableGlobalTimeline = !instance.features.globalTimeline;
	// The backend only has home / local / hybrid / global timelines.
	instance.disableRecommendedTimeline = true;
	instance.softwareVersion = instance.version;
	instance.faviconUrl = instance.iconUrl ?? "";
	if (!Array.isArray(instance.emojis)) instance.emojis = [];
	if (typeof instance.defaultReaction !== "string" || instance.defaultReaction === "") {
		instance.defaultReaction = "⭐";
	}
	if (typeof instance.maxCaptionTextLength !== "number") {
		instance.maxCaptionTextLength = 512;
	}
}

let inflight: Promise<void> | null = null;

/**
 * Fetch meta once and merge it into "instance". Concurrent callers share the
 * in-flight request; pass force = true to refetch (instance info page).
 */
export function updateInstance(force = false): Promise<void> {
	if (inflight != null && !force) return inflight;
	const request = api<MetaDetailed>("meta", { detail: true })
		.then((meta) => {
			Object.assign(instance, meta);
			applyAliases();
			try {
				localStorage.setItem(CACHE_KEY, JSON.stringify(instance));
			} catch {
				// Private mode / quota: the in-memory object is still correct.
			}
			applyInstanceInfo({ name: instance.name, version: instance.version });
		})
		.catch(() => {
			// The instance is unreachable; optimistic defaults stay in place and
			// the account layer surfaces the failure on the first real request.
		})
		.finally(() => {
			inflight = null;
		});
	inflight = request;
	return request;
}

/** Firefish's name for the same operation. */
export const fetchInstance = updateInstance;

export const emojiCategories: ComputedRef<(string | null)[]> = computed(() => {
	const categories = new Set<string | null>();
	for (const emoji of instance.emojis ?? []) categories.add(emoji.category);
	return [...categories];
});

export const emojiTags: ComputedRef<string[]> = computed(() => {
	const tags = new Set<string>();
	for (const emoji of instance.emojis ?? []) {
		for (const alias of emoji.aliases ?? []) tags.add(alias);
	}
	return [...tags];
});
