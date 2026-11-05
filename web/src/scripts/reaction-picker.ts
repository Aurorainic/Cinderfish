// Ported from firefish@b425f61:packages/client/src/scripts/reaction-picker.ts (AGPL-3.0-only)
//
// Degradation: Firefish opens `MkEmojiPickerDialog`, a full picker with
// categories, search and skin tones. That component is not part of this wave, so
// this module renders a compact picker through `os.popup` instead:
//   * the account's default reactions (`defaultStore.state.reactions`)
//   * recently used reactions (`defaultStore.state.recentlyUsedEmojis`)
//   * the instance emoji registry (`api("emojis")`) rendered as images
//   * a text field for a unicode reaction or `:shortcode:`
// The markup is built with `h()` because the panel lives entirely inside this
// file (no component file in the note-core scope) and therefore cannot carry
// scoped SCSS; the inline styles reuse the same CSS variables as the dialogs.
import { defineComponent, h, onMounted, ref, type PropType } from "vue";
import { api } from "@/api/client";
import type { EmojiSimple } from "@/api/types";
import { claimZIndex, popup, type PopupHandle } from "@/os";
import { defaultStore } from "@/store/base";

const MAX_RECENT = 10;
const PANEL_WIDTH = 320;
const PANEL_HEIGHT = 220;

let emojiCache: EmojiSimple[] | null = null;
let emojiRequest: Promise<EmojiSimple[]> | null = null;

/** Custom emoji registry, fetched once and kept for the session. */
export function loadCustomEmojis(): Promise<EmojiSimple[]> {
	if (emojiCache != null) return Promise.resolve(emojiCache);
	if (emojiRequest == null) {
		emojiRequest = api("emojis")
			.then((res) => {
				emojiCache = res.emojis;
				return res.emojis;
			})
			.catch(() => {
				emojiRequest = null;
				return [] as EmojiSimple[];
			});
	}
	return emojiRequest;
}

function pushRecent(reaction: string): void {
	const current = defaultStore.state.recentlyUsedEmojis.filter(
		(x) => x !== reaction,
	);
	defaultStore.set(
		"recentlyUsedEmojis",
		[reaction, ...current].slice(0, MAX_RECENT),
	);
}

function shortcodeUrl(
	reaction: string,
	emojis: EmojiSimple[],
): string | null {
	const match = /^:([^:]+):$/.exec(reaction);
	if (match == null) return null;
	const name = match[1];
	const found = emojis.find((emoji) => emoji.name === name);
	return found?.url ?? null;
}

const itemStyle = {
	display: "inline-flex",
	alignItems: "center",
	justifyContent: "center",
	minInlineSize: "32px",
	blockSize: "32px",
	padding: "2px 4px",
	borderRadius: "6px",
	background: "transparent",
	color: "var(--fg)",
	fontSize: "1.1em",
	cursor: "pointer",
	border: "none",
} as const;

const sectionStyle = {
	display: "flex",
	flexWrap: "wrap",
	gap: "2px",
	alignItems: "center",
} as const;

const ReactionPickerDialog = defineComponent({
	name: "CfReactionPicker",
	props: {
		src: {
			type: Object as PropType<HTMLElement | null>,
			default: null,
		},
	},
	emits: {
		done: (reaction: string) => typeof reaction === "string",
		cancel: () => true,
	},
	setup(props, { emit }) {
		const customEmojis = ref<EmojiSimple[]>([]);
		const text = ref("");
		const zIndex = claimZIndex("high");

		const rect =
			props.src != null ? props.src.getBoundingClientRect() : null;
		const left = Math.max(
			8,
			Math.min(
				rect?.left ?? 8,
				Math.max(8, window.innerWidth - PANEL_WIDTH - 8),
			),
		);
		const top = Math.max(
			8,
			Math.min(
				rect != null ? rect.bottom + 4 : 8,
				Math.max(8, window.innerHeight - PANEL_HEIGHT - 8),
			),
		);

		const panelStyle = {
			position: "fixed",
			top: `${top}px`,
			left: `${left}px`,
			width: `${PANEL_WIDTH}px`,
			maxHeight: `${PANEL_HEIGHT}px`,
			overflowY: "auto",
			zIndex,
			padding: "8px",
			borderRadius: "var(--radius)",
			background: "var(--panel)",
			color: "var(--fg)",
			boxShadow: "var(--shadow)",
			display: "flex",
			flexDirection: "column",
			gap: "8px",
			boxSizing: "border-box",
		} as const;

		onMounted(() => {
			void loadCustomEmojis().then((list) => {
				customEmojis.value = list;
			});
		});

		function choose(reaction: string): void {
			const trimmed = reaction.trim();
			if (trimmed === "") return;
			pushRecent(trimmed);
			emit("done", trimmed);
		}

		function reactionItem(reaction: string) {
			const url = shortcodeUrl(reaction, customEmojis.value);
			return h(
				"button",
				{
					type: "button",
					class: "_button",
					style: itemStyle,
					title: reaction,
					onClick: (ev: MouseEvent) => {
						ev.stopPropagation();
						choose(reaction);
					},
				},
				url != null
					? h("img", {
							src: url,
							alt: reaction,
							style: { inlineSize: "24px", blockSize: "24px" },
					  })
					: reaction,
			);
		}

		return () => {
			const reactions = defaultStore.state.reactions;
			const recent = defaultStore.state.recentlyUsedEmojis;

			const children = [
				h(
					"div",
					{ style: { display: "flex", alignItems: "center", gap: "8px" } },
					[
						h(
							"span",
							{ style: { flexGrow: "1", fontWeight: "bold" } },
							"Reaction",
						),
						h(
							"button",
							{
								type: "button",
								class: "_button",
								style: { ...itemStyle, minInlineSize: "24px", blockSize: "24px" },
								title: "Close",
								onClick: (ev: MouseEvent) => {
									ev.stopPropagation();
									emit("cancel");
								},
							},
							"×",
						),
					],
				),
			];

			if (recent.length > 0) {
				children.push(
					h("div", { style: sectionStyle }, recent.map(reactionItem)),
				);
			}
			children.push(
				h("div", { style: sectionStyle }, reactions.map(reactionItem)),
			);

			if (customEmojis.value.length > 0) {
				children.push(
					h(
						"div",
						{ style: sectionStyle },
						customEmojis.value
							.slice(0, 120)
							.map((emoji) => reactionItem(`:${emoji.name}:`)),
					),
				);
			}

			children.push(
				h("input", {
					class: "cf-reaction-picker-input",
					type: "text",
					value: text.value,
					placeholder: "emoji / :shortcode:",
					style: {
						padding: "6px 8px",
						borderRadius: "var(--radius)",
						border: "solid 1px var(--divider)",
						background: "var(--bg)",
						color: "var(--fg)",
						boxSizing: "border-box",
						inlineSize: "100%",
					},
					onInput: (ev: Event) => {
						text.value = (ev.target as HTMLInputElement).value;
					},
					onKeydown: (ev: KeyboardEvent) => {
						if (ev.key === "Enter") {
							ev.preventDefault();
							choose(text.value);
						} else if (ev.key === "Escape") {
							ev.preventDefault();
							emit("cancel");
						}
					},
				}),
			);

			return h(
				"div",
				{
					class: "cf-reaction-picker",
					style: panelStyle,
					onClick: (ev: MouseEvent) => ev.stopPropagation(),
				},
				children,
			);
		};
	},
});

class ReactionPicker {
	private handle: PopupHandle | null = null;
	private onChosen: ((reaction: string) => void) | null = null;
	private onClosed: (() => void) | null = null;

	public show(
		src: HTMLElement | null | undefined,
		onChosen: (reaction: string) => void,
		onClosed: () => void,
	): void {
		this.close();
		this.onChosen = onChosen;
		this.onClosed = onClosed;
		this.handle = popup(
			ReactionPickerDialog,
			{ src: src ?? null },
			{
				done: (reaction: string) => {
					this.onChosen?.(reaction);
					this.close();
				},
				cancel: () => {
					this.close();
				},
			},
			"closed",
		);
	}

	public close(): void {
		if (this.handle == null) return;
		this.handle.dispose();
		this.handle = null;
		this.onChosen = null;
		const onClosed = this.onClosed;
		this.onClosed = null;
		onClosed?.();
	}
}

export const reactionPicker = new ReactionPicker();
