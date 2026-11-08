<!-- Ported from firefish@b425f61:packages/client/src/components/MkUrlPreview.vue (AGPL-3.0-only) -->
<template>
	<article class="url-preview" @click.stop>
		<a :href="url" rel="nofollow noopener" target="_blank" :title="url">
			<div class="body">
				<h3 :title="url">{{ url }}</h3>
				<p>
					<span>{{ sitename }}</span>
				</p>
			</div>
		</a>
	</article>
</template>

<script setup lang="ts">
import { computed } from "vue";

/**
 * Ported from Firefish with the backend capability gap applied (contract §5):
 * Cinderfish has no /url card endpoint, so there is no title, description,
 * thumbnail, player or embedded tweet to render. The component stays in the
 * tree and renders the plain link -- it never requests /url.
 */
const props = defineProps<{
	url: string;
	detail?: boolean;
	/** Firefish's compact card variant; accepted for call-site parity. */
	compact?: boolean;
}>();

const sitename = computed(() => {
	try {
		return new URL(props.url).hostname;
	} catch {
		return "";
	}
});
</script>

<style lang="scss" scoped>
.url-preview {
	border: 1px solid var(--divider);
	border-radius: var(--radius);
	overflow: hidden;

	> a {
		display: flex;
		padding: 14px 16px;
		text-decoration: none;
		transition: background 0.2s;

		> .body {
			width: 100%;
			min-width: 0;
		}

		h3,
		p {
			display: block;
			margin: 0;
			overflow: hidden;
			text-overflow: ellipsis;
			max-width: 100%;
		}

		h3 {
			font-size: 1em;
			white-space: nowrap;
			margin-bottom: 0.2em;
			text-decoration: underline;
			text-decoration-color: transparent;
			transition: text-decoration-color 0.2s;
		}

		p {
			font-size: 0.8em;
			color: var(--fgTransparentWeak);
		}

		&:hover,
		&:focus,
		&:focus-within {
			background: var(--panelHighlight);

			h3 {
				text-decoration-color: currentColor;
			}
		}
	}
}
</style>