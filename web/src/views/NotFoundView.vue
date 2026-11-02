// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/pages/not-found.vue
//
// The original shows the instance's not-found badge image
// (/static-assets/badges/not-found.webp); Cinderfish serves no static badge
// assets, so the port renders the MkInfo card style with a warning glyph.
<template>
	<div class="not-found">
		<div class="fullinfo">
			<div class="badge" aria-hidden="true">?</div>
			<div>这个地址没有对应的页面。</div>
			<div class="actions">
				<MkButton rounded to="/" link>回到时间线</MkButton>
				<MkButton rounded to="/login" link>登录</MkButton>
			</div>
		</div>
	</div>
</template>

<script lang="ts" setup>
import MkButton from "../components/ui/MkButton.vue";
import { usePageTitle } from "../composables/use-page-title";

usePageTitle("页面不存在");
</script>

<style scoped>
.not-found {
	padding: 32px;
	text-align: center;
}

.fullinfo {
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: 16px;
	padding-block: 64px;
	color: var(--fgTransparentWeak);
}

.badge {
	inline-size: 128px;
	block-size: 128px;
	border-radius: 16px;
	background: var(--accentedBg);
	color: var(--accent);
	font-size: 64px;
	font-weight: bold;
	line-height: 128px;
}

.actions {
	display: flex;
	flex-wrap: wrap;
	justify-content: center;
	gap: 8px;
}
</style>
