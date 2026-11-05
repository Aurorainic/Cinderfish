// Ported from firefish@b425f61:packages/client/src/scripts/autocomplete.ts (AGPL-3.0-only)
//
// Wave-3 degradations (contract §2), all of them enforced by missing endpoints or
// missing dependencies:
//   * `textarea-caret` is not a dependency: the popup anchor is derived from
//     `selectionStart` plus the textarea's computed line height and an average
//     character width. It lags the caret horizontally for proportional fonts and
//     is clamped to the viewport; no pixel-accurate caret following.
//   * user suggestions are gone: Firefish queries
//     `users/search-by-username-and-host`, which Cinderfish does not provide
//     (`users/show` only resolves an exact username), so `@` opens nothing.
//   * hashtag suggestions come from `hashtags/trend` plus the local "hashtags"
//     history instead of `hashtags/search`.
//   * `MkAutocomplete.vue` is not part of this task's write scope, so the list is
//     rendered by a render-function component defined in this file. Its inline
//     styles reuse the same CSS variables as the other os panels.
import { defineComponent, h, onBeforeUnmount, onMounted, ref, watch, type PropType, type Ref } from "vue";
import { popup, type PopupHandle } from "@/os/state";
import { api, resolveMediaUrl } from "@/api/client";
import type { EmojiSimple } from "@/api/types";
import { emojilist } from "@/scripts/emojilist";

type AutocompleteType = "hashtag" | "emoji" | "mfmTag";

interface SuggestionItem {
	/** Text that replaces the trigger sequence. */
	value: string;
	/** Shown in the list. */
	label: string;
	caption?: string;
	icon?: string;
	imageUrl?: string;
}

const MAX_ITEMS = 8;
const PANEL_WIDTH = 260;
const ITEM_HEIGHT = 40;

/** MFM function names, mirrored from `_mfm.*` keys (no endpoint needed). */
const mfmTags = [
	"tada",
	"jelly",
	"bounce",
	"shake",
	"spin",
	"jump",
	"twitch",
	"rainbow",
	"sparkle",
	"blur",
	"x2",
	"x3",
	"x4",
	"center",
	"font",
	"fg.color",
	"bg.color",
	"plain",
	"math",
	"math.block",
	"size",
	"position",
	"scale",
	"rotate",
	"flip",
	"crop",
	"border",
	"button",
	"link",
	"quote",
	"search",
	"dummy",
];

let trendTags: string[] | null = null;
let trendRequest: Promise<string[]> | null = null;
let customEmojis: EmojiSimple[] | null = null;
let emojiRequest: Promise<EmojiSimple[]> | null = null;

async function loadTrendTags(): Promise<string[]> {
	if (trendTags != null) return trendTags;
	if (trendRequest == null) {
		trendRequest = api("hashtags/trend")
			.then((trends) => {
				trendTags = trends.map((trend) => trend.tag);
				return trendTags;
			})
			.catch(() => {
				trendRequest = null;
				return [] as string[];
			});
	}
	return trendRequest;
}

async function loadCustomEmojis(): Promise<EmojiSimple[]> {
	if (customEmojis != null) return customEmojis;
	if (emojiRequest == null) {
		emojiRequest = api("emojis")
			.then((res) => {
				customEmojis = res.emojis ?? [];
				return customEmojis;
			})
			.catch(() => {
				emojiRequest = null;
				return [] as EmojiSimple[];
			});
	}
	return emojiRequest;
}

function hashtagHistory(): string[] {
	try {
		const raw = localStorage.getItem("hashtags");
		if (raw == null) return [];
		const parsed: unknown = JSON.parse(raw);
		return Array.isArray(parsed)
			? parsed.filter((x): x is string => typeof x === "string")
			: [];
	} catch {
		return [];
	}
}

const panelStyle = {
	position: "fixed",
	inlineSize: `${PANEL_WIDTH}px`,
	overflow: "hidden auto",
	padding: "4px",
	borderRadius: "var(--radius)",
	background: "var(--popup)",
	color: "var(--fg)",
	boxShadow: "0 4px 32px var(--shadow)",
	boxSizing: "border-box",
} as const;

const itemStyle = {
	display: "flex",
	alignItems: "center",
	gap: "8px",
	inlineSize: "100%",
	padding: "6px 8px",
	blockSize: `${ITEM_HEIGHT}px`,
	boxSizing: "border-box",
	border: "none",
	borderRadius: "6px",
	background: "transparent",
	color: "inherit",
	font: "inherit",
	fontSize: "0.9em",
	textAlign: "start",
	cursor: "pointer",
} as const;

const AutocompletePopup = defineComponent({
	name: "CfAutocomplete",
	props: {
		x: { type: Object as PropType<Ref<number>>, required: true },
		y: { type: Object as PropType<Ref<number>>, required: true },
		items: { type: Object as PropType<Ref<SuggestionItem[]>>, required: true },
		index: { type: Object as PropType<Ref<number>>, required: true },
	},
	emits: {
		done: (_item: SuggestionItem) => true,
		cancel: () => true,
	},
	setup(props, { emit }) {
		watch(
			() => props.items.value,
			() => {
				props.index.value = 0;
			},
		);

		function choose(item: SuggestionItem | undefined): void {
			if (item == null) return;
			emit("done", item);
		}

		function onKeydown(event: KeyboardEvent): void {
			const items = props.items.value;
			if (items.length === 0) return;

			if (event.key === "ArrowDown") {
				event.preventDefault();
				props.index.value = (props.index.value + 1) % items.length;
			} else if (event.key === "ArrowUp") {
				event.preventDefault();
				props.index.value =
					(props.index.value - 1 + items.length) % items.length;
			} else if (event.key === "Enter" || event.key === "Tab") {
				event.preventDefault();
				choose(items[props.index.value]);
			} else if (event.key === "Escape") {
				event.preventDefault();
				emit("cancel");
			}
		}

		let panelEl: HTMLElement | null = null;

		function onPointerDown(event: PointerEvent): void {
			const target = event.target as Node | null;
			if (target != null && panelEl != null && !panelEl.contains(target)) {
				emit("cancel");
			}
		}

		onMounted(() => {
			window.addEventListener("keydown", onKeydown, true);
			window.addEventListener("pointerdown", onPointerDown, true);
		});

		onBeforeUnmount(() => {
			window.removeEventListener("keydown", onKeydown, true);
			window.removeEventListener("pointerdown", onPointerDown, true);
		});

		function row(item: SuggestionItem, i: number) {
			return h(
				"button",
				{
					type: "button",
					style: {
						...itemStyle,
						background:
							i === props.index.value ? "var(--listItemHoverBg)" : "transparent",
					},
					onMousedown: (event: MouseEvent) => event.preventDefault(),
					onClick: () => choose(item),
				},
				[
					item.imageUrl != null
						? h("img", {
								src: resolveMediaUrl(item.imageUrl),
								alt: item.label,
								style: {
									inlineSize: "22px",
									blockSize: "22px",
									objectFit: "contain",
								},
						  })
						: item.icon != null
							? h("i", { class: item.icon })
							: null,
					h(
						"span",
						{
							style: {
								flex: "1",
								overflow: "hidden",
								textOverflow: "ellipsis",
								whiteSpace: "nowrap",
							},
						},
						item.label,
					),
					item.caption != null
						? h(
								"span",
								{ style: { opacity: "0.6", fontSize: "0.85em" } },
								item.caption,
						  )
						: null,
				],
			);
		}

		return () => {
			const items = props.items.value;
			const height = Math.min(items.length * ITEM_HEIGHT + 8, 320);
			const x = Math.max(
				8,
				Math.min(props.x.value, window.innerWidth - PANEL_WIDTH - 8),
			);
			const y = Math.max(
				8,
				Math.min(props.y.value, window.innerHeight - height - 8),
			);

			return h(
				"div",
				{
					ref: (el: unknown) => {
						panelEl = el as HTMLElement | null;
					},
					class: "cf-autocomplete _shadow",
					style: { ...panelStyle, top: `${y}px`, left: `${x}px`, maxBlockSize: `${height}px` },
					onMousedown: (event: MouseEvent) => event.preventDefault(),
				},
				items.map(row),
			);
		};
	},
});

export class Autocomplete {
	private suggestion: {
		x: Ref<number>;
		y: Ref<number>;
		items: Ref<SuggestionItem[]>;
		index: Ref<number>;
		close: () => void;
	} | null = null;
	private textarea: HTMLInputElement | HTMLTextAreaElement;
	private currentType: AutocompleteType | null = null;
	private textRef: Ref<string>;
	private opening = false;
	private suppressInput = false;

	private get text(): string {
		// Raw .value, because v-model does not update during IME composition.
		return this.textarea.value;
	}

	constructor(
		textarea: HTMLInputElement | HTMLTextAreaElement | null,
		textRef: Ref<string>,
	) {
		this.textarea = textarea ?? document.createElement("textarea");
		this.textRef = textRef;
		if (textarea != null) this.attach();
	}

	public attach(): void {
		this.textarea.addEventListener("input", this.onInput);
		this.textarea.addEventListener("blur", this.onBlur);
	}

	public detach(): void {
		this.textarea.removeEventListener("input", this.onInput);
		this.textarea.removeEventListener("blur", this.onBlur);
		this.close();
	}

	private onInput = (): void => {
		if (this.suppressInput) {
			this.suppressInput = false;
			return;
		}
		void this.onInputImpl();
	};

	private onBlur = (): void => {
		// A click inside the popup keeps the suggestion open; anything else closes.
		window.setTimeout(() => {
			this.close();
		}, 150);
	};

	private async onInputImpl(): Promise<void> {
		const caretPos = this.textarea.selectionStart ?? this.text.length;
		const line = this.text.substring(0, caretPos).split("\n").pop() ?? "";

		const hashtagIndex = line.lastIndexOf("#");
		const emojiIndex = line.lastIndexOf(":");
		const mfmTagIndex = line.lastIndexOf("$");

		const isMfmTag = mfmTagIndex > hashtagIndex && mfmTagIndex > emojiIndex;
		const isHashtag = hashtagIndex > emojiIndex && !isMfmTag;
		const isEmoji =
			emojiIndex !== -1 &&
			emojiIndex > mfmTagIndex &&
			line.split(/:[a-z0-9_+\-]+:/).pop()?.includes(":") === true;

		if (isMfmTag) {
			const tag = line.substring(mfmTagIndex + 1).replace("[", "");
			if (!tag.includes(" ")) {
				await this.open("mfmTag", tag);
				return;
			}
		} else if (isHashtag) {
			const tag = line.substring(hashtagIndex + 1);
			if (!tag.includes(" ")) {
				await this.open("hashtag", tag);
				return;
			}
		} else if (isEmoji) {
			const emoji = line.substring(emojiIndex + 1);
			if (!emoji.includes(" ")) {
				await this.open("emoji", emoji);
				return;
			}
		}

		this.close();
	}

	private items(type: AutocompleteType, query: string): SuggestionItem[] {
		const q = query.replace(/^:/, "").toLowerCase();

		if (type === "hashtag") {
			const tags = Array.from(
				new Set([...hashtagHistory(), ...(trendTags ?? [])]),
			);
			return tags
				.filter((tag) => q === "" || tag.toLowerCase().startsWith(q))
				.slice(0, MAX_ITEMS)
				.map((tag) => ({
					value: tag,
					label: `#${tag}`,
					icon: "ph-hash ph-bold ph-lg",
				}));
		}

		if (type === "mfmTag") {
			return mfmTags
				.filter((tag) => q === "" || tag.startsWith(q))
				.slice(0, MAX_ITEMS)
				.map((tag) => ({
					value: tag,
					label: `$[${tag}`,
					icon: "ph-magic-wand ph-bold ph-lg",
				}));
		}

		const custom = (customEmojis ?? [])
			.filter(
				(emoji) =>
					q === "" ||
					emoji.name.toLowerCase().startsWith(q) ||
					emoji.aliases.some((alias) => alias.toLowerCase().startsWith(q)),
			)
			.slice(0, MAX_ITEMS)
			.map((emoji) => ({
				value: `:${emoji.name}:`,
				label: `:${emoji.name}:`,
				imageUrl: emoji.url,
			}));

		if (custom.length >= MAX_ITEMS) return custom;

		const unicode = emojilist
			.filter(
				(emoji) =>
					q === "" ||
					emoji.slug.startsWith(q) ||
					emoji.keywords.some((keyword) => keyword.startsWith(q)),
			)
			.slice(0, MAX_ITEMS - custom.length)
			.map((emoji) => ({
				value: emoji.emoji,
				label: emoji.emoji,
				caption: emoji.slug,
			}));

		return [...custom, ...unicode];
	}

	private caretPoint(): { x: number; y: number } {
		const el = this.textarea;
		const style = window.getComputedStyle(el);
		const fontSize = Number.parseFloat(style.fontSize) || 14;
		const lineHeight = Number.parseFloat(style.lineHeight) || fontSize * 1.4;
		const paddingLeft = Number.parseFloat(style.paddingLeft) || 0;
		const paddingTop = Number.parseFloat(style.paddingTop) || 0;

		const caretPos = el.selectionStart ?? el.value.length;
		const lines = el.value.substring(0, caretPos).split("\n");
		const row = lines.length - 1;
		const column = lines[row].length;

		const rect = el.getBoundingClientRect();
		return {
			x: rect.left + paddingLeft + column * fontSize * 0.6 - el.scrollLeft,
			y:
				rect.top +
				paddingTop +
				(row + 1) * lineHeight -
				el.scrollTop,
		};
	}

	private async open(type: AutocompleteType, query: string): Promise<void> {
		if (type !== this.currentType) this.close();
		this.currentType = type;

		if (type === "hashtag") await loadTrendTags();
		if (type === "emoji") await loadCustomEmojis();

		const items = this.items(type, query);

		if (this.suggestion == null) {
			if (this.opening) return;
			this.opening = true;

			const { x, y } = this.caretPoint();
			const xRef = ref(x);
			const yRef = ref(y);
			const itemsRef = ref(items);
			const indexRef = ref(0);

			const handle: PopupHandle = popup(
				AutocompletePopup,
				{ x: xRef, y: yRef, items: itemsRef, index: indexRef },
				{
					done: (item: SuggestionItem) => {
						this.complete(type, item.value);
					},
					cancel: () => {
						this.close();
					},
				},
				"closed",
			);

			this.suggestion = {
				x: xRef,
				y: yRef,
				items: itemsRef,
				index: indexRef,
				close: () => handle.dispose(),
			};
			this.opening = false;
			return;
		}

		this.suggestion.items.value = items;
		const { x, y } = this.caretPoint();
		this.suggestion.x.value = x;
		this.suggestion.y.value = y;
	}

	private close(): void {
		if (this.suggestion == null) return;
		this.suggestion.close();
		this.suggestion = null;
		this.currentType = null;
	}

	private complete(type: AutocompleteType, value: string): void {
		const el = this.textarea;
		const caret = el.selectionStart ?? el.value.length;
		const before = el.value.substring(0, caret);
		const trigger = type === "hashtag" ? "#" : type === "emoji" ? ":" : "$";
		const triggerIndex = before.lastIndexOf(trigger);

		this.close();

		if (triggerIndex < 0) return;

		const insert =
			type === "hashtag"
				? `#${value} `
				: type === "emoji"
					? value
					: `$[${value} ]`;

		this.suppressInput = true;
		el.setRangeText(insert, triggerIndex, caret, "end");
		this.textRef.value = el.value;
		el.dispatchEvent(new Event("input", { bubbles: true }));
		el.focus();
	}
}
