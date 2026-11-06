<!-- Ported from firefish@b425f61:packages/client/src/pages/not-found.vue (AGPL-3.0-only) -->
<template>
	<div class="not-found">
		<div class="_fullinfo">
			<i class="ph-ghost ph-bold icon"></i>
			<div class="code">404</div>
			<div class="description">{{ t("notFoundDescription") }}</div>
			<div class="actions">
				<CfButton primary rounded link to="/">
					<i class="ph-house ph-bold ph-lg"></i>
					<span>{{ t("home") }}</span>
				</CfButton>
				<CfButton rounded link to="/explore">
					<i class="ph-compass ph-bold ph-lg"></i>
					<span>{{ t("explore") }}</span>
				</CfButton>
			</div>
			<p class="hint">页面可能已被删除，或者链接有误。</p>
		</div>
	</div>
</template>

<script lang="ts" setup>
// Firefish renders /static-assets/badges/not-found.png here. That badge is not
// shipped by Cinderfish (web/public/static-assets/badges/ only has info.svg),
// so the illustration is a Phosphor glyph — same degradation MkError applies to
// the error badge.
import CfButton from "@/components/ui/CfButton.vue";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageNotFound" });

definePageMetadata({
	title: t("notFound"),
	icon: "ph-warning ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.not-found {
	inline-size: 100%;
	max-inline-size: 800px;
	margin-inline: auto;
	padding: 16px;
	box-sizing: border-box;

	._fullinfo {
		display: flex;
		flex-direction: column;
		align-items: center;
	}

	> ._fullinfo > .icon {
		font-size: 72px;
		opacity: 0.4;
		margin-block-end: 8px;
	}

	> ._fullinfo > .code {
		font-size: 2em;
		font-weight: 800;
		letter-spacing: 0.06em;
	}

	> ._fullinfo > .description {
		margin-block-start: 4px;
		color: var(--fgTransparentWeak);
	}

	> ._fullinfo > .actions {
		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: 8px;
		margin-block-start: 24px;
	}

	> ._fullinfo > .hint {
		margin-block-start: 16px;
		font-size: 0.85em;
		color: var(--fgTransparentWeak);
	}
}
</style>
