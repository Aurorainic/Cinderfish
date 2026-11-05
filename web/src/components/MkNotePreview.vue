<!-- Ported from firefish@b425f61:packages/client/src/components/MkNotePreview.vue (AGPL-3.0-only) -->
<template>
	<div v-if="author != null" v-size="{ min: [350, 500] }" class="fefdfafb">
		<MkAvatar class="avatar" :user="author" disableLink />
		<div class="main">
			<div class="header">
				<MkUserName :user="author" />
			</div>
			<div class="body">
				<div class="content">
					<Mfm :text="text.trim()" :author="author" :i="author" advancedMfm />
				</div>
			</div>
		</div>
	</div>
</template>

<script lang="ts" setup>
// Degradation: Firefish runs the text through `preprocess()`, which only expands
// custom KaTeX macros. katex is not a dependency of this port, so the raw text is
// rendered as-is.
import { computed } from "vue";
import type { MeDetailed } from "@/api/types";
import { me } from "@/account";

defineProps<{
	text: string;
}>();

const author = computed<MeDetailed | null>(() => me.value);
</script>

<style lang="scss" scoped>
.fefdfafb {
	display: flex;
	margin: 0;
	padding: 0;
	overflow: clip;
	font-size: 0.95em;

	&.min-width_350px {
		> .avatar {
			margin: 0 10px 0 0;
			width: 44px;
			height: 44px;
		}
	}

	&.min-width_500px {
		> .avatar {
			margin: 0 12px 0 0;
			width: 48px;
			height: 48px;
		}
	}

	> .avatar {
		flex-shrink: 0;
		display: block;
		margin: 0 10px 0 0;
		width: 40px;
		height: 40px;
		border-radius: 8px;
		pointer-events: none;
	}

	> .main {
		flex: 1;
		min-width: 0;

		> .header {
			margin-bottom: 2px;
			font-weight: bold;
		}

		> .body {
			> .cw {
				cursor: default;
				display: block;
				margin: 0;
				padding: 0;
				overflow-wrap: break-word;

				> .text {
					margin-right: 8px;
				}
			}

			> .content {
				> .text {
					cursor: default;
					margin: 0;
					padding: 0;
				}
			}
		}
	}
}
</style>
