/**
 * Port of Firefish `packages/client/src/cold-store.ts`.
 *
 * Device-wide preferences that do not need reactivity: sound effects, media
 * volume, plugin data. Firefish keeps the `miux:` localStorage prefix; we keep
 * it too so an existing browser profile carries over.
 */

const PREFIX = "miux:";

export interface SoundConfig {
	type: string;
	volume: number;
}

export interface PluginRecord {
	id: string;
	name: string;
	active: boolean;
	configData: Record<string, unknown>;
	token: string;
	ast: unknown[];
}

export interface ColdStoreShape {
	mediaVolume: number;
	vibrate: boolean;
	sound_masterVolume: number;
	sound_note: SoundConfig;
	sound_noteMy: SoundConfig;
	sound_notification: SoundConfig;
	sound_chat: SoundConfig;
	sound_chatBg: SoundConfig;
	sound_antenna: SoundConfig;
	sound_channel: SoundConfig;
	plugins: PluginRecord[];
}

const defaults: ColdStoreShape = {
	mediaVolume: 0.5,
	vibrate: false,
	sound_masterVolume: 0.3,
	sound_note: { type: "none", volume: 0 },
	sound_noteMy: { type: "syuilo/up", volume: 1 },
	sound_notification: { type: "syuilo/pope2", volume: 1 },
	sound_chat: { type: "syuilo/pope1", volume: 1 },
	sound_chatBg: { type: "syuilo/waon", volume: 1 },
	sound_antenna: { type: "syuilo/triple", volume: 1 },
	sound_channel: { type: "syuilo/square-pico", volume: 1 },
	plugins: [],
};

const watchers: {
	key: string;
	callback: (value: unknown) => void;
}[] = [];

export function get<K extends keyof ColdStoreShape>(key: K): ColdStoreShape[K] {
	const value = localStorage.getItem(PREFIX + key);
	if (value == null) return defaults[key];
	try {
		return JSON.parse(value) as ColdStoreShape[K];
	} catch {
		return defaults[key];
	}
}

export function set<K extends keyof ColdStoreShape>(
	key: K,
	value: ColdStoreShape[K],
): void {
	if (value === undefined) {
		console.error(`attempt to store undefined value for key '${key}'`);
		return;
	}
	localStorage.setItem(PREFIX + key, JSON.stringify(value));
	for (const watcher of watchers) {
		if (watcher.key === key) watcher.callback(value);
	}
}

export function watch<K extends keyof ColdStoreShape>(
	key: K,
	callback: (value: ColdStoreShape[K]) => void,
): void {
	watchers.push({ key, callback: callback as (value: unknown) => void });
}

export function makeGetterSetter<K extends keyof ColdStoreShape>(
	key: K,
): { get: () => ColdStoreShape[K]; set: (value: ColdStoreShape[K]) => void } {
	return {
		get: () => get(key),
		set: (value) => set(key, value),
	};
}

export const ColdDeviceStorage = {
	get,
	set,
	watch,
	makeGetterSetter,
	defaults,
};
