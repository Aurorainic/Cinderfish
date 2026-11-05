<!-- Ported from firefish@b425f61:packages/client/src/components/MkCheatSheetDialog.vue (AGPL-3.0-only) -->
<template>
	<CfModal ref="modal" :close-on-backdrop="true" @closed="emit('closed')">
		<div class="_popup _shadow panel" @click.stop>
			<header>
				<span class="title">{{ t("_mfm.cheatSheet") }}</span>
				<button class="_button close" :aria-label="t('close')" @click="close">
					<i class="ph-x ph-bold ph-lg"></i>
				</button>
			</header>
			<div class="body">
				<p class="intro">{{ t("_mfm.intro") }}</p>
				<div v-for="entry in entries" :key="entry.syntax" class="entry">
					<code>{{ entry.syntax }}</code>
					<div class="text">
						<span class="name">{{ t(entry.title) }}</span>
						<span class="description">{{ t(entry.description) }}</span>
					</div>
				</div>
			</div>
		</div>
	</CfModal>
</template>

<script lang="ts" setup>
// Degradation (contract §2): Firefish renders `pages/mfm-cheat-sheet.vue` inside
// a window, and that page renders each MFM function with the real Mfm component.
// katex is not a dependency here, so `$[math]` entries are shown as source code
// only, and the reference list is a compact table built from the `_mfm.*`
// translation keys instead of importing the page (kept out of this scope).
import { ref } from "vue";
import CfModal from "@/os/CfModal.vue";
import { t } from "@/i18n";

const emit = defineEmits<{
	done: [];
	closed: [];
}>();

const modal = ref<InstanceType<typeof CfModal> | null>(null);

interface CheatSheetEntry {
	syntax: string;
	title: string;
	description: string;
}

const entries: CheatSheetEntry[] = [
	{ syntax: "**text**", title: "_mfm.bold", description: "_mfm.boldDescription" },
	{ syntax: "$[small text]", title: "_mfm.small", description: "_mfm.smallDescription" },
	{ syntax: "> text", title: "_mfm.quote", description: "_mfm.quoteDescription" },
	{ syntax: "`code`", title: "_mfm.inlineCode", description: "_mfm.inlineCodeDescription" },
	{ syntax: "```\ncode\n```", title: "_mfm.blockCode", description: "_mfm.blockCodeDescription" },
	{ syntax: "$[math] a^2 + b^2 $\n$[math.block] \\int_0^1 x dx $[/math]", title: "_mfm.inlineMath", description: "_mfm.inlineMathDescription" },
	{ syntax: "$[math.block] \\int_0^1 x dx $[/math]", title: "_mfm.blockMath", description: "_mfm.blockMathDescription" },
	{ syntax: ":name:", title: "_mfm.emoji", description: "_mfm.emojiDescription" },
	{ syntax: "@user", title: "_mfm.mention", description: "_mfm.mentionDescription" },
	{ syntax: "#tag", title: "_mfm.hashtag", description: "_mfm.hashtagDescription" },
	{ syntax: "[title](https://example.com)", title: "_mfm.link", description: "_mfm.linkDescription" },
	{ syntax: "https://example.com", title: "_mfm.url", description: "_mfm.urlDescription" },
	{ syntax: "$[tada text]", title: "_mfm.tada", description: "_mfm.tadaDescription" },
	{ syntax: "$[jelly text]", title: "_mfm.jelly", description: "_mfm.jellyDescription" },
	{ syntax: "$[bounce text]", title: "_mfm.bounce", description: "_mfm.bounceDescription" },
	{ syntax: "$[shake text]", title: "_mfm.shake", description: "_mfm.shakeDescription" },
	{ syntax: "$[spin text]", title: "_mfm.spin", description: "_mfm.spinDescription" },
	{ syntax: "$[jump text]", title: "_mfm.jump", description: "_mfm.jumpDescription" },
	{ syntax: "$[twitch text]", title: "_mfm.twitch", description: "_mfm.twitchDescription" },
	{ syntax: "$[rainbow text]", title: "_mfm.rainbow", description: "_mfm.rainbowDescription" },
	{ syntax: "$[sparkle text]", title: "_mfm.sparkle", description: "_mfm.sparkleDescription" },
	{ syntax: "$[blur text]", title: "_mfm.blur", description: "_mfm.blurDescription" },
	{ syntax: "$[x2 text]", title: "_mfm.x2", description: "_mfm.x2Description" },
	{ syntax: "$[x3 text]", title: "_mfm.x3", description: "_mfm.x3Description" },
	{ syntax: "$[x4 text]", title: "_mfm.x4", description: "_mfm.x4Description" },
	{ syntax: "$[center text]", title: "_mfm.center", description: "_mfm.centerDescription" },
	{ syntax: "$[font.serif text]", title: "_mfm.font", description: "_mfm.fontDescription" },
	{ syntax: "$[fg.color=ff0000 text]", title: "_mfm.foreground", description: "_mfm.foregroundDescription" },
	{ syntax: "$[bg.color=ff0000 text]", title: "_mfm.background", description: "_mfm.backgroundDescription" },
	{ syntax: "$[plain **text**]", title: "_mfm.plain", description: "_mfm.plainDescription" },
];

function close(): void {
	emit("done");
	modal.value?.close();
	emit("closed");
}
</script>

<style lang="scss" scoped>
.panel {
	inline-size: min(600px, 100%);
	max-block-size: calc(100dvb - 64px);
	display: flex;
	flex-direction: column;
	background: var(--panel);
	border-radius: var(--radius);
	box-shadow: 0 4px 32px var(--shadow);
	overflow: hidden;

	> header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 14px 16px;
		border-block-end: solid 0.5px var(--divider);

		> .title {
			font-weight: bold;
		}

		> .close {
			padding: 4px;
			color: var(--fg);
		}
	}

	> .body {
		padding: 8px 16px 16px;
		overflow-y: auto;

		> .intro {
			margin: 8px 0 16px;
			font-size: 0.85em;
			color: var(--fgTransparentWeak);
			line-height: 1.6;
		}

		> .entry {
			padding: 10px 0;
			border-block-end: solid 0.5px var(--divider);

			> code {
				display: block;
				margin-block-end: 4px;
				padding: 4px 8px;
				border-radius: 6px;
				background: var(--X3);
				font-size: 0.85em;
				white-space: pre-wrap;
				overflow-wrap: anywhere;
			}

			> .text {
				display: flex;
				flex-direction: column;
				font-size: 0.85em;

				> .name {
					font-weight: bold;
				}

				> .description {
					opacity: 0.7;
					line-height: 1.5;
				}
			}
		}
	}
}
</style>
