// Ported from firefish@b425f61:packages/client/src/directives/hotkey.ts (AGPL-3.0-only)
import type { Directive } from "vue";
import { makeHotkey } from "@/scripts/hotkey";

type HotkeyBinding = Record<string, ((ev: KeyboardEvent) => void) | undefined>;

type Handler = (ev: KeyboardEvent) => void;

const handlers = new WeakMap<HTMLElement, { handler: Handler; global: boolean }>();

export default {
	mounted(el: HTMLElement, binding: { value: HotkeyBinding; modifiers: Record<string, boolean> }) {
		const handler = makeHotkey(binding.value);
		const global = binding.modifiers.global === true;
		handlers.set(el, { handler, global });
		if (global) document.addEventListener("keydown", handler);
		else el.addEventListener("keydown", handler);
	},

	updated(el: HTMLElement, binding: { value: HotkeyBinding; modifiers: Record<string, boolean> }) {
		const previous = handlers.get(el);
		if (previous == null) return;
		if (previous.global) document.removeEventListener("keydown", previous.handler);
		else el.removeEventListener("keydown", previous.handler);
		const handler = makeHotkey(binding.value);
		const global = binding.modifiers.global === true;
		handlers.set(el, { handler, global });
		if (global) document.addEventListener("keydown", handler);
		else el.addEventListener("keydown", handler);
	},

	unmounted(el: HTMLElement) {
		const previous = handlers.get(el);
		if (previous == null) return;
		if (previous.global) document.removeEventListener("keydown", previous.handler);
		else el.removeEventListener("keydown", previous.handler);
		handlers.delete(el);
	},
} as Directive<HTMLElement, HotkeyBinding>;
