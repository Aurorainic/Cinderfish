<!-- Ported from firefish@b425f61:packages/client/src/ui/universal.vue (AGPL-3.0-only) -->
<template>
	<div class="dkgtipfy" :class="{ isMobile }">
		<XSidebar v-if="!isMobile" />

		<main id="maincontent" class="contents">
			<RouterView />
		</main>

		<div v-if="isDesktop" class="widgets-container">
			<MkWidgets />
		</div>

		<nav v-if="isMobile" class="buttons">
			<button
				class="button _button"
				:aria-label="t('menu')"
				@click="drawerShowing = true"
			>
				<i class="ph-list ph-bold ph-lg"></i>
			</button>
			<CfA to="/" class="button" active-class="active" :aria-label="t('timeline')">
				<i class="ph-house ph-bold ph-lg"></i>
			</CfA>
			<CfA
				to="/my/notifications"
				class="button"
				active-class="active"
				:aria-label="t('notifications')"
			>
				<i class="ph-bell ph-bold ph-lg"></i>
				<span v-if="notificationIndicated" class="indicator">
					<i class="ph-circle ph-fill"></i>
				</span>
			</CfA>
			<CfA to="/explore" class="button" active-class="active" :aria-label="t('explore')">
				<i class="ph-compass ph-bold ph-lg"></i>
			</CfA>
			<CfA to="/my/drive" class="button" active-class="active" :aria-label="t('drive')">
				<i class="ph-cloud ph-bold ph-lg"></i>
			</CfA>
			<CfA to="/settings" class="button" active-class="active" :aria-label="t('settings')">
				<i class="ph-gear-six ph-bold ph-lg"></i>
			</CfA>
		</nav>

		<button
			v-if="isMobile"
			class="postButton button _button"
			:aria-label="t('note')"
			data-cy-open-post-form
			@click="openPost"
		>
			<i class="ph-pencil ph-bold ph-lg"></i>
		</button>

		<transition name="drawer-back">
			<div
				v-if="drawerShowing"
				class="drawer-back _modalBg"
				@click="drawerShowing = false"
			></div>
		</transition>
		<transition name="drawer">
			<aside v-if="drawerShowing" class="drawer">
				<MkNavbar drawer />
			</aside>
		</transition>
	</div>
</template>

<script setup lang="ts">
// Firefish's universal shell is a flex row: the navbar column, the main column
// (which owns the page's scroll) and, on wide screens, a sticky widgets column.
// A bottom button bar replaces the navbar below 500px.
//
// DEGRADED (see the wave-3 contract §2):
//   * `XStatusBars` has no Cinderfish port; the sticky header row is omitted.
//   * Firefish's widgets *drawer* (for 500px–1090px) is dropped — the column is
//     shown when there is room and hidden otherwise, so nothing is fabricated.
//   * The right-click "open in window" context menu is left to `CfA`.
import { computed, onUnmounted, ref, watch, type Ref } from "vue";
import { useRoute } from "vue-router";
import { me } from "@/account";
import { useLocale } from "@/i18n";
import CfA from "@/components/ui/CfA.vue";
import MkNavbar from "@/components/MkNavbar.vue";
import MkWidgets from "@/components/MkWidgets.vue";
import { post } from "@/os/post";
import XSidebar from "./XSidebar.vue";

const DESKTOP_THRESHOLD = 1100;
const MOBILE_THRESHOLD = 500;

function useMediaFlag(query: string): Ref<boolean> {
	const flag = ref(
		typeof window === "undefined" ? false : window.matchMedia(query).matches,
	);
	if (typeof window !== "undefined") {
		const media = window.matchMedia(query);
		const onChange = (event: MediaQueryListEvent): void => {
			flag.value = event.matches;
		};
		media.addEventListener("change", onChange);
		onUnmounted(() => media.removeEventListener("change", onChange));
	}
	return flag;
}

const isMobile = useMediaFlag(`(max-width: ${MOBILE_THRESHOLD}px)`);
const isDesktop = useMediaFlag(`(min-width: ${DESKTOP_THRESHOLD}px)`);

const route = useRoute();
const { t } = useLocale();

const drawerShowing = ref(false);
watch(
	() => route.fullPath,
	() => {
		drawerShowing.value = false;
	},
);

const notificationIndicated = computed(
	() =>
		me.value != null &&
		(me.value.unreadNotificationsCount > 0 || me.value.hasUnreadNotification),
);

function openPost(): void {
	void post();
}
</script>

<style lang="scss" scoped>
.dkgtipfy {
	display: flex;
	min-block-size: 100vh;
	box-sizing: border-box;
	background: var(--bg);

	> .contents {
		flex: 1;
		min-inline-size: 0;
		box-sizing: border-box;
		overflow-x: clip;
	}

	&.isMobile > .contents {
		padding-block-end: calc(env(safe-area-inset-bottom, 0px) + 96px);
	}

	&:not(.isMobile) > .contents {
		border-inline-end: solid 0.5px var(--divider);
	}

	> .widgets-container {
		position: sticky;
		inset-block-start: 0;
		flex: 0 0 300px;
		inline-size: 300px;
		max-block-size: 100vh;
		padding: 0 var(--margin);
		box-sizing: content-box;
		overflow-y: auto;
	}

	> .buttons {
		position: fixed;
		z-index: 1000;
		inset-block-end: 0;
		inset-inline: 0;
		display: flex;
		inline-size: 100%;
		padding: 12px 12px calc(env(safe-area-inset-bottom, 0px) + 12px) 12px;
		box-sizing: border-box;
		background: var(--bg);

		> .button {
			position: relative;
			flex: 1;
			display: flex;
			align-items: center;
			justify-content: center;
			block-size: 3.5rem;
			margin: auto;
			padding: 0;
			border-radius: 8px;
			color: var(--fg);
			transition: background 0.1s;

			&:not(:last-child) {
				margin-inline-end: 12px;
			}

			&:active,
			&.active {
				color: var(--accent);
				background: var(--accentedBg);
			}

			> .indicator {
				position: absolute;
				inset-block-start: 0;
				inset-inline-start: 0;
				color: var(--navIndicator);
				font-size: 16px;
			}
		}
	}

	> .postButton {
		position: fixed;
		z-index: 1000;
		inset-block-end: calc(env(safe-area-inset-bottom, 0px) + 6rem);
		inset-inline-end: 1.5rem;
		inline-size: 4rem;
		block-size: 4rem;
		color: var(--fg);
		background: var(--panelHighlight);
		border-radius: 10px;
		box-shadow: var(--shadow) 0 0 25px;
		transition:
			background 0.6s,
			transform 0.3s;

		&:active {
			background: var(--accentedBg);
		}
	}

	> .drawer-back {
		z-index: 1001;
	}

	> .drawer {
		position: fixed;
		inset-block: 0;
		inset-inline-start: 0;
		z-index: 1001;
		inline-size: 240px;
		block-size: 100%;
		box-sizing: border-box;
		overflow: auto;
		overscroll-behavior: contain;
		background: var(--navBg);
	}
}

.drawer-enter-active,
.drawer-leave-active {
	opacity: 1;
	transform: translateX(0);
	transition:
		transform 300ms cubic-bezier(0.23, 1, 0.32, 1),
		opacity 300ms cubic-bezier(0.23, 1, 0.32, 1);
}

.drawer-enter-from,
.drawer-leave-to {
	opacity: 0;
	transform: translateX(-240px);
}

.drawer-back-enter-active,
.drawer-back-leave-active {
	opacity: 1;
	transition: opacity 300ms cubic-bezier(0.23, 1, 0.32, 1);
}

.drawer-back-enter-from,
.drawer-back-leave-to {
	opacity: 0;
}
</style>
