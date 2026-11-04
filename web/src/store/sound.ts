import { createPizzax } from "./pizzax";

/**
 * Sound preferences. Firefish keeps these in a non-reactive ColdStore bucket
 * with the `miux:` prefix; they are device-scoped preferences in practice, so
 * this port puts them in a normal pizzax bucket while keeping the key names
 * (`sound_masterVolume`, `sound_notification`, ...).
 */

export interface SoundSettings {
	type: string;
	volume: number;
}

export const soundStore = createPizzax("sound", {
	masterVolume: { where: "device", default: 0.3 },
	note: { where: "device", default: { type: "none", volume: 0 } as SoundSettings },
	noteMy: {
		where: "device",
		default: { type: "syuilo/up", volume: 1 } as SoundSettings,
	},
	notification: {
		where: "device",
		default: { type: "syuilo/pope2", volume: 1 } as SoundSettings,
	},
	chat: {
		where: "device",
		default: { type: "syuilo/pope1", volume: 1 } as SoundSettings,
	},
	chatBg: {
		where: "device",
		default: { type: "syuilo/waon", volume: 1 } as SoundSettings,
	},
	antenna: {
		where: "device",
		default: { type: "syuilo/triple", volume: 1 } as SoundSettings,
	},
	channel: {
		where: "device",
		default: { type: "syuilo/square-pico", volume: 1 } as SoundSettings,
	},
});
