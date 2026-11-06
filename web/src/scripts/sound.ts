// Ported from firefish@b425f61:packages/client/src/scripts/sound.ts (AGPL-3.0-only)
//
// Firefish plays `/static-assets/sounds/<type>.mp3`. The syuilo set is copied
// from the reference checkout into web/public/static-assets/sounds/syuilo/
// (see docs/HANDOVER.md §12FK for the provenance record), so playback is on.
import { ColdDeviceStorage } from "@/store";

const SOUND_ASSETS_AVAILABLE = true;

/** The `sound_*` cold-store keys Firefish's callers use. */
export type SoundType =
	| "note"
	| "noteMy"
	| "notification"
	| "chat"
	| "chatBg"
	| "antenna"
	| "channel";

const cache = new Map<string, HTMLAudioElement>();

export function getAudio(file: string, useCache = true): HTMLAudioElement {
	let audio: HTMLAudioElement;
	if (useCache && cache.has(file)) {
		audio = cache.get(file) as HTMLAudioElement;
	} else {
		audio = new Audio(`/static-assets/sounds/${file}.mp3`);
		if (useCache) cache.set(file, audio);
	}
	return audio;
}

export function setVolume(
	audio: HTMLAudioElement,
	volume: number,
): HTMLAudioElement {
	const masterVolume = ColdDeviceStorage.get("sound_masterVolume");
	audio.volume = masterVolume - (1 - volume) * masterVolume;
	return audio;
}

export function play(type: SoundType): void {
	// The cold store is still honoured (settings pages edit it), only playback is
	// disabled while the assets are missing.
	const sound = ColdDeviceStorage.get(`sound_${type}`);
	if (sound.type == null) return;
	playFile(sound.type, sound.volume);
}

export function playFile(file: string, volume: number): void {
	const masterVolume = ColdDeviceStorage.get("sound_masterVolume");
	if (masterVolume === 0 || volume === 0 || file.toLowerCase().includes("none"))
		return;
	if (!SOUND_ASSETS_AVAILABLE) return;

	const audio = setVolume(getAudio(file), volume);
	void audio.play().catch(() => {
		// Autoplay policy or a missing asset: Firefish sound effects are optional.
	});
}
