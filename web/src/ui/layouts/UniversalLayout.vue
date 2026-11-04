<template>
	<div class="dkgtipfy" :class="{ isMobile }">
		<XSidebar v-if="!isMobile" />

		<main id="maincontent" class="contents">
			<router-view />
		</main>

		<div v-if="isDesktop" class="widgets-container">
			<aside class="widgets">
				<div class="widget-placeholder">TODO: port widgets</div>
			</aside>
		</div>

		<nav v-if="isMobile" class="mobile-nav">
			<RouterLink to="/" class="item" :class="{ active: isRoute('/') }">
				<span class="icon">⌂</span>
			</RouterLink>
			<RouterLink to="/my/notifications" class="item">
				<span class="icon">🔔</span>
			</RouterLink>
			<RouterLink to="/explore" class="item">
				<span class="icon">⌕</span>
			</RouterLink>
			<RouterLink to="/my/drive" class="item">
				<span class="icon">▤</span>
			</RouterLink>
			<RouterLink to="/settings" class="item">
				<span class="icon">⚙</span>
			</RouterLink>
		</nav>
	</div>
</template>

<script setup lang="ts">
import { useRoute } from "vue-router";
import XSidebar from "./XSidebar.vue";
import { useLayout } from "@/router/layout";

const { isMobile, isDesktop } = useLayout();
const route = useRoute();

function isRoute(path: string): boolean {
	return route.path === path;
}
</script>

<style scoped>
.dkgtipfy {
	display: flex;
	min-block-size: 100dvh;
	background: var(--bg);
	color: var(--fg);

	> .contents {
		flex: 1;
		min-inline-size: 0;
		box-sizing: border-box;
	}

	> .widgets-container {
		inline-size: 330px;
		flex-shrink: 0;
		padding: 16px 16px 16px 0;
		box-sizing: border-box;
	}

	> .widgets {
		position: sticky;
		inset-block-start: 16px;
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
}

.widget-placeholder {
	padding: 16px;
	color: var(--fgTransparentWeak);
	text-align: center;
	background: var(--panel);
	border-radius: var(--radius, 12px);
}

.mobile-nav {
	position: fixed;
	inset-inline: 0;
	inset-block-end: 0;
	z-index: 500;
	display: flex;
	justify-content: space-around;
	background: var(--header);
	-webkit-backdrop-filter: var(--blur, blur(15px));
	backdrop-filter: var(--blur, blur(15px));
	border-block-start: solid 0.5px var(--divider);

	> .item {
		display: flex;
		align-items: center;
		justify-content: center;
		flex: 1;
		padding-block: 12px;
		color: var(--fgTransparentWeak);

		&.active {
			color: var(--accent);
		}
	}
}
</style>
