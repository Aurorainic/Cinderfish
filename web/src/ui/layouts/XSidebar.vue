<template>
	<aside class="x-sidebar">
		<div class="brand">
			<RouterLink to="/" class="brand-link">Cinderfish</RouterLink>
		</div>
		<nav class="nav">
			<RouterLink to="/" class="nav-item" :class="{ active: route.path === '/' }">
				<span class="icon">⌂</span><span class="label">{{ t("timeline") }}</span>
			</RouterLink>
			<RouterLink to="/my/notifications" class="nav-item">
				<span class="icon">🔔</span><span class="label">{{ t("notifications") }}</span>
			</RouterLink>
			<RouterLink to="/explore" class="nav-item">
				<span class="icon">⌕</span><span class="label">{{ t("explore") }}</span>
			</RouterLink>
			<RouterLink to="/my/favorites" class="nav-item">
				<span class="icon">★</span><span class="label">{{ t("favorites") }}</span>
			</RouterLink>
			<RouterLink to="/my/drive" class="nav-item">
				<span class="icon">▤</span><span class="label">{{ t("drive") }}</span>
			</RouterLink>
			<RouterLink to="/settings" class="nav-item">
				<span class="icon">⚙</span><span class="label">{{ t("settings") }}</span>
			</RouterLink>
		</nav>
		<div class="bottom">
			<CfAccountButton />
		</div>
	</aside>
</template>

<script setup lang="ts">
import { useRoute } from "vue-router";
import { translate as t } from "@/i18n";
import CfAccountButton from "@/account/CfAccountButton.vue";

const route = useRoute();
</script>

<style scoped>
.x-sidebar {
	position: sticky;
	inset-block-start: 0;
	display: flex;
	flex-direction: column;
	inline-size: 230px;
	block-size: 100dvh;
	flex-shrink: 0;
	padding: 16px 8px;
	box-sizing: border-box;
	border-inline-end: solid 0.5px var(--divider);
	background: var(--navBg, var(--panel));

	> .brand {
		padding-block: 8px 16px;
		padding-inline: 12px;
		font-size: 1.25em;
		font-weight: bold;
	}

	> .nav {
		display: flex;
		flex-direction: column;
		gap: 2px;

		> .nav-item {
			display: flex;
			align-items: center;
			gap: 12px;
			padding: 10px 12px;
			color: var(--navFg);
			border-radius: var(--radius, 12px);
			text-decoration: none;

			&:hover {
				color: var(--navHoverFg);
				background: var(--buttonBg);
			}

			&.active {
				color: var(--navActive);
				background: var(--accentedBg);
				font-weight: bold;
			}

			> .icon {
				inline-size: 20px;
				text-align: center;
			}
		}
	}

	> .bottom {
		margin-block-start: auto;
	}
}
</style>
