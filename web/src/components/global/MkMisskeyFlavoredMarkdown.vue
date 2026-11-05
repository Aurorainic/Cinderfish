<!-- Ported from firefish@b425f61:packages/client/src/components/global/MkMisskeyFlavoredMarkdown.vue (AGPL-3.0-only) -->
<!-- Ported from firefish@b425f61:packages/client/src/components/mfm.ts (AGPL-3.0-only) -->
<script lang="ts">
// Firefish keeps the MFM renderer in components/mfm.ts (defineComponent + render)
// and wraps it in MkMisskeyFlavoredMarkdown.vue for the store flags and styles.
// Cinderfish merges both into this single component; the render-function form is
// kept because the parser walk is written against "this" props, and the style
// block moved to styles/mfm.scss as the wave-2 contract requires.
import { defineComponent, h, type PropType, type VNode } from "vue";
import * as mfm from "mfm-js";
import MkUrl from "./MkUrl.vue";
import MkLink from "../MkLink.vue";
import MkMention from "../MkMention.vue";
import MkEmoji from "./MkEmoji.vue";
import { concat } from "../../scripts/array";
import MkFormula from "../MkFormula.vue";
import MkCode from "../MkCode.vue";
import MkGoogle from "../MkGoogle.vue";
import MkSparkle from "../MkSparkle.vue";
import MkA from "./MkA.vue";
import { host } from "../../config";
import { reducedMotion } from "../../scripts/reduced-motion";
import { defaultStore } from "../../store";
import type { EmojiSimple, UserLite } from "../../api/types";

type MfmChild = VNode | string;

function timeArg(value: string | true | undefined): string | null {
	if (typeof value !== "string") return null;
	return /^[0-9.]+s$/.test(value) ? value : null;
}

function numberArg(value: string | true | undefined): string | null {
	if (typeof value !== "string") return null;
	const parsed = Number.parseFloat(value);
	return !Number.isNaN(parsed) && Number.isFinite(parsed) && parsed > 0
		? value
		: null;
}

function floatArg(value: string | true | undefined, fallback: number): number {
	if (typeof value !== "string") return fallback;
	const parsed = Number.parseFloat(value);
	return Number.isFinite(parsed) ? parsed : fallback;
}

function colorArg(value: string | true | undefined): string {
	const color = typeof value === "string" ? value : "";
	return /^[0-9a-f]{3,6}$/i.test(color) ? color : "f00";
}

const blockTypes = ["blockCode", "center", "mathBlock", "quote", "search"];

export default defineComponent({
	name: "Mfm",

	props: {
		text: {
			type: String,
			required: true,
		},
		plain: {
			type: Boolean,
			default: false,
		},
		nowrap: {
			type: Boolean,
			default: false,
		},
		author: {
			type: Object as PropType<UserLite | null>,
			default: null,
		},
		i: {
			type: Object,
			default: null,
		},
		customEmojis: {
			type: Array as PropType<EmojiSimple[]>,
			required: false,
			default: undefined,
		},
		isNote: {
			type: Boolean,
			default: true,
		},
		/** Firefish keeps this on the wrapper component; kept optional here. */
		advancedMfm: {
			type: Boolean,
			default: false,
		},
	},

	render(): VNode | void {
		if (this.text == null || this.text === "") return;

		const isPlain = this.plain;

		let ast: mfm.MfmNode[];
		try {
			ast = (isPlain ? mfm.parseSimple : mfm.parse)(this.text);
		} catch (err) {
			// Malformed input must never break the note tree.
			console.error("failed to parse MFM:", err);
			return h("span", this.text);
		}

		const animatedMfm = defaultStore.state.animatedMfm;
		const advancedMfm = defaultStore.state.advancedMfm || this.advancedMfm;
		// Firefish disables the inline animation/transform styles through a
		// stylesheet override; without "!important" the same decision is taken
		// here, before the style string is built.
		const suppress = !animatedMfm && !advancedMfm;

		const genEl = (nodes: mfm.MfmNode[]): MfmChild[] =>
			concat(
				nodes.map((token, index): MfmChild[] => {
					switch (token.type) {
						case "text": {
							const text = token.props.text.replace(/\r\n|\n|\r/g, "\n");

							if (!this.plain) {
								const res: MfmChild[] = [];
								for (const line of text.split("\n")) {
									res.push(h("br"));
									res.push(line);
								}
								res.shift();
								return res;
							}
							return [text.replace(/\n/g, " ")];
						}

						case "bold": {
							return [h("b", genEl(token.children))];
						}

						case "strike": {
							return [h("del", genEl(token.children))];
						}

						case "italic": {
							return [
								h("i", { style: "font-style: oblique;" }, genEl(token.children)),
							];
						}

						case "fn": {
							const fnName = token.props.name;
							const args = token.props.args;
							const children = genEl(token.children);
							let style: string | null = null;
							let recognised = true;

							switch (fnName) {
								case "tada": {
									const speed = timeArg(args.speed) || "1s";
									const delay = timeArg(args.delay) || "0s";
									const loop = numberArg(args.loop) || "infinite";
									if (!suppress) {
										style =
											"font-size: 150%; animation: tada " +
											speed +
											" " +
											delay +
											" linear " +
											loop +
											" both;";
									}
									break;
								}
								case "jelly": {
									const speed = timeArg(args.speed) || "1s";
									const delay = timeArg(args.delay) || "0s";
									const loop = numberArg(args.loop) || "infinite";
									if (!suppress) {
										style =
											"animation: mfm-rubberBand " +
											speed +
											" " +
											delay +
											" linear " +
											loop +
											" both;";
									}
									break;
								}
								case "twitch": {
									const speed = timeArg(args.speed) || "0.5s";
									const delay = timeArg(args.delay) || "0s";
									const loop = numberArg(args.loop) || "infinite";
									if (!suppress) {
										style =
											"animation: mfm-twitch " +
											speed +
											" " +
											delay +
											" ease " +
											loop +
											";";
									}
									break;
								}
								case "shake": {
									const speed = timeArg(args.speed) || "0.5s";
									const delay = timeArg(args.delay) || "0s";
									const loop = numberArg(args.loop) || "infinite";
									if (!suppress) {
										style =
											"animation: mfm-shake " +
											speed +
											" " +
											delay +
											" ease " +
											loop +
											";";
									}
									break;
								}
								case "spin": {
									const direction = args.left
										? "reverse"
										: args.alternate
										? "alternate"
										: "normal";
									const anime = args.x
										? "mfm-spinX"
										: args.y
										? "mfm-spinY"
										: "mfm-spin";
									const speed = timeArg(args.speed) || "1.5s";
									const delay = timeArg(args.delay) || "0s";
									const loop = numberArg(args.loop) || "infinite";
									if (!suppress) {
										style =
											"animation: " +
											anime +
											" " +
											speed +
											" " +
											delay +
											" linear " +
											loop +
											"; animation-direction: " +
											direction +
											";";
									}
									break;
								}
								case "jump": {
									const speed = timeArg(args.speed) || "0.75s";
									const delay = timeArg(args.delay) || "0s";
									const loop = numberArg(args.loop) || "infinite";
									if (!suppress) {
										style =
											"animation: mfm-jump " +
											speed +
											" " +
											delay +
											" linear " +
											loop +
											";";
									}
									break;
								}
								case "bounce": {
									const speed = timeArg(args.speed) || "0.75s";
									const delay = timeArg(args.delay) || "0s";
									const loop = numberArg(args.loop) || "infinite";
									if (!suppress) {
										style =
											"animation: mfm-bounce " +
											speed +
											" " +
											delay +
											" linear " +
											loop +
											"; transform-origin: center bottom;";
									}
									break;
								}
								case "rainbow": {
									const speed = timeArg(args.speed) || "1s";
									const delay = timeArg(args.delay) || "0s";
									const loop = numberArg(args.loop) || "infinite";
									if (!suppress) {
										style =
											"animation: mfm-rainbow " +
											speed +
											" " +
											delay +
											" linear " +
											loop +
											";";
									}
									break;
								}
								case "sparkle": {
									if (reducedMotion()) return children;
									return [h(MkSparkle, {}, () => children)];
								}
								case "fade": {
									const direction = args.out
										? "alternate-reverse"
										: "alternate";
									const speed = timeArg(args.speed) || "1.5s";
									const delay = timeArg(args.delay) || "0s";
									const loop = numberArg(args.loop) || "infinite";
									if (!suppress) {
										style =
											"animation: mfm-fade " +
											speed +
											" " +
											delay +
											" linear " +
											loop +
											"; animation-direction: " +
											direction +
											";";
									}
									break;
								}
								case "flip": {
									if (!suppress) {
										const transform =
											args.h && args.v
												? "scale(-1, -1)"
												: args.v
												? "scaleY(-1)"
												: "scaleX(-1)";
										style = "transform: " + transform + ";";
									}
									break;
								}
								case "x2": {
									return [h("span", { class: "mfm-x2" }, children)];
								}
								case "x3": {
									return [h("span", { class: "mfm-x3" }, children)];
								}
								case "x4": {
									return [h("span", { class: "mfm-x4" }, children)];
								}
								case "font": {
									const family = args.serif
										? "serif"
										: args.monospace
										? "monospace"
										: args.cursive
										? "cursive"
										: args.fantasy
										? "fantasy"
										: args.emoji
										? "emoji"
										: args.math
										? "math"
										: null;
									if (family && !suppress) {
										style = "font-family: " + family + ";";
									}
									break;
								}
								case "blur": {
									return [h("span", { class: "_blur_text" }, children)];
								}
								case "rotate": {
									if (!suppress) {
										const rotate = args.x
											? "perspective(128px) rotateX"
											: args.y
											? "perspective(128px) rotateY"
											: "rotate";
										const degrees = Number.parseInt(
											typeof args.deg === "string" ? args.deg : "",
											10,
										);
										style =
											"transform: " +
											rotate +
											"(" +
											(Number.isNaN(degrees) ? 90 : degrees) +
											"deg); transform-origin: center center;";
									}
									break;
								}
								case "position": {
									if (!suppress) {
										const x = floatArg(args.x, 0);
										const y = floatArg(args.y, 0);
										style =
											"transform: translateX(" +
											x +
											"em) translateY(" +
											y +
											"em);";
									}
									break;
								}
								case "crop": {
									if (!suppress) {
										const top = floatArg(args.top, 0);
										const right = floatArg(args.right, 0);
										const bottom = floatArg(args.bottom, 0);
										const left = floatArg(args.left, 0);
										style =
											"clip-path: inset(" +
											top +
											"% " +
											right +
											"% " +
											bottom +
											"% " +
											left +
											"%);";
									}
									break;
								}
								case "scale": {
									if (!suppress) {
										const x = Math.min(floatArg(args.x, 1), 5);
										const y = Math.min(floatArg(args.y, 1), 5);
										style = "transform: scale(" + x + ", " + y + ");";
									}
									break;
								}
								case "fg": {
									if (!suppress) {
										style = "color: #" + colorArg(args.color) + ";";
									}
									break;
								}
								case "bg": {
									if (!suppress) {
										style =
											"background-color: #" + colorArg(args.color) + ";";
									}
									break;
								}
								case "small": {
									return [
										h("small", { style: "opacity: 0.7;" }, children),
									];
								}
								case "center": {
									return [
										h("div", { style: "text-align: center;" }, children),
									];
								}
								default: {
									recognised = false;
									break;
								}
							}

							if (!recognised) {
								return [
									h("span", {}, [
										"$[",
										fnName,
										" ",
										...children,
										"]",
									]),
								];
							}
							if (style == null) return [h("span", {}, children)];
							return [
								h(
									"span",
									{ style: "display: inline-block;" + style },
									children,
								),
							];
						}

						case "small": {
							return [
								h(
									"small",
									{ style: "opacity: 0.7;" },
									genEl(token.children),
								),
							];
						}

						case "center": {
							return [
								h(
									"div",
									{ style: "text-align: center;" },
									genEl(token.children),
								),
							];
						}

						case "url": {
							return [
								h(MkUrl, {
									key: "url" + index,
									url: token.props.url,
									rel: "nofollow noopener",
								}),
							];
						}

						case "link": {
							return [
								h(
									MkLink,
									{
										key: "link" + index,
										url: token.props.url,
										rel: "nofollow noopener",
									},
									() => genEl(token.children),
								),
							];
						}

						case "mention": {
							const author = this.author;
							return [
								h(MkMention, {
									key: "mention" + index,
									host:
										(token.props.host == null &&
										author != null &&
										author.host != null
											? author.host
											: token.props.host) || host,
									username: token.props.username,
								}),
							];
						}

						case "hashtag": {
							return [
								h(
									MkA,
									{
										key: "hashtag" + index,
										to:
											"/tags/" +
											encodeURIComponent(token.props.hashtag),
										style: "color:var(--hashtag);",
									},
									() => ["#" + token.props.hashtag],
								),
							];
						}

						case "blockCode": {
							return [
								h(MkCode, {
									key: "blockCode" + index,
									code: token.props.code,
									lang: token.props.lang ?? undefined,
								}),
							];
						}

						case "inlineCode": {
							return [
								h(MkCode, {
									key: "inlineCode" + index,
									code: token.props.code,
									inline: true,
								}),
							];
						}

						case "quote": {
							if (!this.nowrap) {
								return [h("blockquote", genEl(token.children))];
							}
							return [
								h("span", { class: "quote" }, genEl(token.children)),
							];
						}

						case "emojiCode": {
							return [
								h(MkEmoji, {
									key: "emojiCode" + index,
									emoji: ":" + token.props.name + ":",
									customEmojis: this.customEmojis,
									normal: this.plain,
								}),
							];
						}

						case "unicodeEmoji": {
							return [
								h(MkEmoji, {
									key: "unicodeEmoji" + index,
									emoji: token.props.emoji,
									customEmojis: this.customEmojis,
									normal: this.plain,
								}),
							];
						}

						case "mathInline": {
							return [
								h(MkFormula, {
									key: "mathInline" + index,
									formula: token.props.formula,
									block: false,
								}),
							];
						}

						case "mathBlock": {
							return [
								h(MkFormula, {
									key: "mathBlock" + index,
									formula: token.props.formula,
									block: true,
								}),
							];
						}

						case "search": {
							// Disable the "search" keyword (see the issue #9816 on Codeberg).
							if (
								token.props.content.slice(-6).toLowerCase() === "search"
							) {
								const sentinel = "#";
								const ast2 = (isPlain ? mfm.parseSimple : mfm.parse)(
									token.props.content.slice(0, -6) + sentinel,
								);
								const last = ast2[ast2.length - 1];
								if (
									last != null &&
									last.type === "text" &&
									last.props.text.endsWith(sentinel)
								) {
									last.props.text = last.props.text.slice(0, -1);
								} else {
									// I don't think this scope is reachable
									console.warn(
										"Something went wrong while parsing MFM. Please send a bug report, if possible.",
									);
								}

								let prefix = "\n";
								const previous = index === 0 ? null : ast[index - 1];
								if (
									previous == null ||
									blockTypes.includes(previous.type)
								) {
									prefix = "";
								}

								return [
									prefix,
									...genEl(ast2),
									token.props.content.slice(-6) + "\n",
								];
							}

							return [
								h(MkGoogle, {
									key: "search" + index,
									q: token.props.query,
								}),
							];
						}

						case "plain": {
							return [h("span", genEl(token.children))];
						}

						default: {
							// Unknown node: render its raw text instead of throwing, so a
							// newer mfm-js parser can never break a note.
							const unknown = token as unknown as { type: string };
							console.error("unrecognized ast type:", unknown.type);
							return [];
						}
					}
				}),
			);

		return h(
			"span",
			{
				class: [
					"mfm-object",
					this.nowrap ? "nowrap" : null,
					advancedMfm ? "advancedMfm" : null,
				],
			},
			genEl(ast),
		);
	},
});
</script>
