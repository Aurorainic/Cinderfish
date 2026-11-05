<!-- Ported from firefish@b425f61:packages/client/src/pages/mfm-cheat-sheet.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader v-if="!popup" :actions="headerActions" />
		</template>
		<MkSpacer :content-max="800">
			<div class="mfm-cheat-sheet">
				<p class="intro">{{ t("_mfm.intro") }}</p>

				<section
					v-for="section in sections"
					:key="section.id"
					class="section _block"
				>
					<div class="title">{{ section.title }}</div>
					<div class="content">
						<p v-if="section.description">
							{{ section.description }}
						</p>
						<div class="preview">
							<div class="rendered">
								<Mfm :text="preview[section.id]" />
							</div>
							<CfTextarea v-model="preview[section.id]" code>
								<template #label>MFM</template>
							</CfTextarea>
						</div>
					</div>
				</section>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Firefish's cheat sheet is a hand-written list of ~35 sections; the same list
// is kept here, but generated from one definition array so the title, the
// description and the editable sample stay in a single row.
//
// Contract §2 degradations:
//   * katex is not installed, so $[math] renders as code (MkFormula) instead of
//     typeset math — the math sections are still listed because the nodes are
//     supported by the renderer;
//   * `unicode-emoji-json` is not installed, so the unicode emoji section uses
//     a fixed sample instead of a picker;
//   * the custom emoji sample cannot ask the instance for a real name here, so
//     it falls back to `:emojiname:` exactly like Firefish does on an instance
//     without registered emoji.
// The section list mirrors the node types handled by
// components/global/MkMisskeyFlavoredMarkdown.vue: text, bold, italic, strike,
// small, center, big (x2/x3/x4), font, blur, quote, code (inline/block), math
// (inline/block), url, link, mention, hashtag, emojiCode, unicodeEmoji,
// animations (fn), flip, rotate, position, crop, scale, fg, bg, search, plain.
import { computed, ref } from "vue";
import CfTextarea from "@/components/ui/CfTextarea.vue";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageMfmCheatSheet" });

defineProps<{ popup?: boolean }>();

const dummy = t("_mfm.dummy");

interface MfmSection {
	id: string;
	title: string;
	description: string;
}

const sections = computed<MfmSection[]>(() => [
	{
		id: "mention",
		title: t("_mfm.mention"),
		description: t("_mfm.mentionDescription"),
	},
	{
		id: "hashtag",
		title: t("_mfm.hashtag"),
		description: t("_mfm.hashtagDescription"),
	},
	{
		id: "url",
		title: t("_mfm.url"),
		description: t("_mfm.urlDescription"),
	},
	{
		id: "link",
		title: t("_mfm.link"),
		description: t("_mfm.linkDescription"),
	},
	{
		id: "emoji",
		title: t("_mfm.emoji"),
		description: t("_mfm.emojiDescription"),
	},
	{
		id: "unicodeEmoji",
		title: "Unicode 表情",
		description: "直接输入的 Unicode 表情会按原样显示。",
	},
	{
		id: "bold",
		title: t("_mfm.bold"),
		description: t("_mfm.boldDescription"),
	},
	{
		id: "strike",
		title: "删除线",
		description: "使用 ~~ 包裹文字来添加删除线。",
	},
	{
		id: "italic",
		title: "斜体",
		description: "使用 * 包裹文字来显示斜体。",
	},
	{
		id: "small",
		title: t("_mfm.small"),
		description: t("_mfm.smallDescription"),
	},
	{
		id: "center",
		title: t("_mfm.center"),
		description: t("_mfm.centerDescription"),
	},
	{
		id: "quote",
		title: t("_mfm.quote"),
		description: t("_mfm.quoteDescription"),
	},
	{
		id: "inlineCode",
		title: t("_mfm.inlineCode"),
		description: t("_mfm.inlineCodeDescription"),
	},
	{
		id: "blockCode",
		title: t("_mfm.blockCode"),
		description: t("_mfm.blockCodeDescription"),
	},
	{
		id: "inlineMath",
		title: t("_mfm.inlineMath"),
		description: `${t("_mfm.inlineMathDescription")}（Cinderfish 未引入 katex，公式以代码显示）`,
	},
	{
		id: "blockMath",
		title: t("_mfm.blockMath"),
		description: `${t("_mfm.blockMathDescription")}（Cinderfish 未引入 katex，公式以代码显示）`,
	},
	{
		id: "flip",
		title: t("_mfm.flip"),
		description: t("_mfm.flipDescription"),
	},
	{
		id: "font",
		title: t("_mfm.font"),
		description: t("_mfm.fontDescription"),
	},
	{
		id: "x2",
		title: t("_mfm.x2"),
		description: t("_mfm.x2Description"),
	},
	{
		id: "x3",
		title: t("_mfm.x3"),
		description: t("_mfm.x3Description"),
	},
	{
		id: "x4",
		title: t("_mfm.x4"),
		description: t("_mfm.x4Description"),
	},
	{
		id: "blur",
		title: t("_mfm.blur"),
		description: t("_mfm.blurDescription"),
	},
	{
		id: "jelly",
		title: t("_mfm.jelly"),
		description: t("_mfm.jellyDescription"),
	},
	{
		id: "tada",
		title: t("_mfm.tada"),
		description: t("_mfm.tadaDescription"),
	},
	{
		id: "jump",
		title: t("_mfm.jump"),
		description: t("_mfm.jumpDescription"),
	},
	{
		id: "bounce",
		title: t("_mfm.bounce"),
		description: t("_mfm.bounceDescription"),
	},
	{
		id: "shake",
		title: t("_mfm.shake"),
		description: t("_mfm.shakeDescription"),
	},
	{
		id: "twitch",
		title: t("_mfm.twitch"),
		description: t("_mfm.twitchDescription"),
	},
	{
		id: "spin",
		title: t("_mfm.spin"),
		description: t("_mfm.spinDescription"),
	},
	{
		id: "rainbow",
		title: t("_mfm.rainbow"),
		description: t("_mfm.rainbowDescription"),
	},
	{
		id: "sparkle",
		title: t("_mfm.sparkle"),
		description: t("_mfm.sparkleDescription"),
	},
	{
		id: "rotate",
		title: t("_mfm.rotate"),
		description: t("_mfm.rotateDescription"),
	},
	{
		id: "fade",
		title: t("_mfm.fade"),
		description: t("_mfm.fadeDescription"),
	},
	{
		id: "crop",
		title: t("_mfm.crop"),
		description: t("_mfm.cropDescription"),
	},
	{
		id: "position",
		title: t("_mfm.position"),
		description: t("_mfm.positionDescription"),
	},
	{
		id: "scale",
		title: t("_mfm.scale"),
		description: t("_mfm.scaleDescription"),
	},
	{
		id: "fg",
		title: t("_mfm.foreground"),
		description: t("_mfm.foregroundDescription"),
	},
	{
		id: "bg",
		title: t("_mfm.background"),
		description: t("_mfm.backgroundDescription"),
	},
	{
		id: "search",
		title: t("_mfm.search"),
		description: t("_mfm.searchDescription"),
	},
	{
		id: "plain",
		title: t("_mfm.plain"),
		description: t("_mfm.plainDescription"),
	},
]);

// Samples are edited live: each textarea writes back into this map, which is
// what Firefish's per-section `v-model` did through the reactivity transform.
const preview = ref<Record<string, string>>({
	mention: "@example",
	hashtag: "#test",
	url: "https://cinderfish.example",
	link: `[${dummy}](https://cinderfish.example)`,
	emoji: ":emojiname:",
	unicodeEmoji: "😀 👍 🎉",
	bold: `**${dummy}**`,
	strike: `~~${dummy}~~`,
	italic: `*${dummy}*`,
	small: `<small>${dummy}</small> $[small ${dummy}]`,
	center: `<center>${dummy}</center> $[center ${dummy}]`,
	quote: `> ${dummy}`,
	inlineCode: '`<: "Hello, world!"`',
	blockCode:
		"```\n~ (#i, 100) {\n\t<: ? ((i % 15) = 0) \"FizzBuzz\"\n\t\t.? ((i % 3) = 0) \"Fizz\"\n\t\t.? ((i % 5) = 0) \"Buzz\"\n\t\t. i\n}\n```",
	inlineMath: "\\(x= \\frac{-b' \\pm \\sqrt{(b')^2-ac}}{a}\\)",
	blockMath: "\\[x= \\frac{-b' \\pm \\sqrt{(b')^2-ac}}{a}\\]",
	flip: `$[flip ${dummy}]\n$[flip.v ${dummy}]\n$[flip.h,v ${dummy}]`,
	font: `$[font.serif ${dummy}]\n$[font.monospace ${dummy}]\n$[font.cursive ${dummy}]\n$[font.fantasy ${dummy}]`,
	x2: "$[x2 🍮]",
	x3: "$[x3 🍮]",
	x4: "$[x4 🍮]",
	blur: `$[blur ${dummy}]`,
	jelly: "$[jelly 🍮] $[jelly.speed=3s 🍮] $[jelly.delay=3s 🍮] $[jelly.loop=3 🍮]",
	tada: "$[tada 🍮] $[tada.speed=3s 🍮] $[tada.delay=3s 🍮] $[tada.loop=3 🍮]",
	jump: "$[jump 🍮] $[jump.speed=3s 🍮] $[jump.delay=3s 🍮] $[jump.loop=3 🍮]",
	bounce: "$[bounce 🍮] $[bounce.speed=3s 🍮] $[bounce.delay=3s 🍮] $[bounce.loop=3 🍮]",
	shake: "$[shake 🍮] $[shake.speed=3s 🍮] $[shake.delay=3s 🍮] $[shake.loop=3 🍮]",
	twitch: "$[twitch 🍮] $[twitch.speed=3s 🍮] $[twitch.delay=3s 🍮] $[twitch.loop=3 🍮]",
	spin: "$[spin 🍮] $[spin.left 🍮] $[spin.alternate 🍮]\n$[spin.x 🍮] $[spin.x,left 🍮] $[spin.x,alternate 🍮]\n$[spin.y 🍮] $[spin.y,left 🍮] $[spin.y,alternate 🍮]\n\n$[spin.speed=3s 🍮] $[spin.delay=3s 🍮] $[spin.loop=3 🍮]",
	rainbow:
		"$[rainbow 🍮] $[rainbow.speed=3s 🍮] $[rainbow.delay=3s 🍮] $[rainbow.loop=3 🍮]",
	sparkle: "$[sparkle 🍮]",
	rotate: "$[rotate 🍮]\n$[rotate.deg=45 🍮]\n$[rotate.x,deg=45 Hello, world!]",
	fade: "$[fade 🍮] $[fade.out 🍮] $[fade.speed=3s 🍮] $[fade.delay=3s 🍮]",
	crop: "$[crop.top=50 🍮] $[crop.right=50 🍮] $[crop.bottom=50 🍮] $[crop.left=50 🍮]",
	position: "$[position.y=-1 🍮]\n$[position.x=-1 🍮]",
	scale: "$[scale.x=1.3 🍮]\n$[scale.x=1.5,y=3 🍮]\n$[scale.y=0.3 🍮]",
	fg: "$[fg.color=eb6f92 Text color]",
	bg: "$[bg.color=31748f Background color]",
	search: `${dummy} [search]`,
	plain: `<plain>**bold** @mention #hashtag \`code\` $[x2 🍮]</plain>`,
});

const headerActions = computed(() => []);

definePageMetadata({
	title: t("_mfm.cheatSheet"),
	icon: "ph-question ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.mfm-cheat-sheet {
	> .intro {
		margin: 0;
		padding: 16px;
	}

	> .section {
		> .title {
			position: sticky;
			z-index: 1;
			top: var(--stickyTop, 0px);
			padding: 16px;
			font-weight: bold;
			-webkit-backdrop-filter: var(--blur, blur(10px));
			backdrop-filter: var(--blur, blur(10px));
			background-color: var(--X16);
		}

		> .content {
			> p {
				margin: 0;
				padding: 16px;
				padding-top: 0;
			}

			> .preview {
				border-top: solid 0.5px var(--divider);
				padding: 16px;
				display: flex;
				flex-direction: column;
				gap: 12px;

				> .rendered {
					min-block-size: 24px;
					overflow-wrap: anywhere;
				}
			}
		}
	}
}
</style>
