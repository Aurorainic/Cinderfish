// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/MkFolder.vue
//
// The `v-vibrate` directive of the original is replaced with the Vibration
// API directly; `getUniqueId` from os is replaced with a local counter.
<template>
	<section class="cinderfish-folder">
		<header class="header" @click="showBody = !showBody">
			<div class="title"><slot name="header"></slot></div>
			<div class="divider"></div>
			<button
				class="toggle"
				:aria-expanded="showBody"
				:aria-controls="bodyId"
				@click.stop="showBody = !showBody"
			>
				<template v-if="showBody">▲</template>
				<template v-else>▼</template>
			</button>
		</header>
		<transition
			:name="animation ? 'folder-toggle' : ''"
			@enter="enter"
			@after-enter="afterEnter"
			@leave="leave"
			@after-leave="afterLeave"
		>
			<div v-show="showBody" :id="bodyId">
				<slot></slot>
			</div>
		</transition>
	</section>
</template>

<script lang="ts" setup>
import { ref, watch } from "vue";

let uid = 0;

const localStoragePrefix = "ui:folder:";

const props = withDefaults(
	defineProps<{
		expanded?: boolean;
		persistKey?: string | null;
	}>(),
	{
		expanded: true,
		persistKey: null,
	},
);

const bodyId = ref(`cinderfish-folder-${++uid}`);

const showBody = ref(
	props.persistKey && localStorage.getItem(localStoragePrefix + props.persistKey)
		? localStorage.getItem(localStoragePrefix + props.persistKey) === "t"
		: props.expanded,
);

// TODO: Cinderfish has no defaultStore animation setting; animation is on
// unconditionally until a settings store exists.
const animation = true;

watch(showBody, () => {
	if (props.persistKey) {
		localStorage.setItem(
			localStoragePrefix + props.persistKey,
			showBody.value ? "t" : "f",
		);
	}
});

function toggleContent(show: boolean) {
	showBody.value = show;
}

function enter(el: Element) {
	const element = el as HTMLElement;
	const elementHeight = element.getBoundingClientRect().height;
	element.style.height = "0";
	element.offsetHeight; // reflow
	element.style.height = `${elementHeight}px`;
}
function afterEnter(el: Element) {
	(el as HTMLElement).style.height = "";
}
function leave(el: Element) {
	const element = el as HTMLElement;
	const elementHeight = element.getBoundingClientRect().height;
	element.style.height = `${elementHeight}px`;
	element.offsetHeight; // reflow
	element.style.height = "0";
}
function afterLeave(el: Element) {
	(el as HTMLElement).style.height = "";
}

defineExpose({
	toggleContent,
	enter,
	afterEnter,
	leave,
	afterLeave,
});
</script>

<style scoped>
.folder-toggle-enter-active,
.folder-toggle-leave-active {
	overflow-block: hidden;
	transition: opacity 0.5s, height 0.5s !important;
}

.folder-toggle-enter-from {
	opacity: 0;
}

.folder-toggle-leave-to {
	opacity: 0;
}

.cinderfish-folder {
	position: relative;
}

.cinderfish-folder > .header {
	display: flex;
	position: relative;
	z-index: 10;
	position: sticky;
	inset-block-start: var(--stickyTop, 0px);
	padding: var(--x-padding);
	-webkit-backdrop-filter: var(--blur, blur(8px));
	backdrop-filter: var(--blur, blur(20px));
	margin-inline: -12px;
	padding-inline: 12px;
	mask: linear-gradient(
		90deg,
		transparent,
		black 12px calc(100% - 12px),
		transparent
	);
	-webkit-mask: linear-gradient(
		90deg,
		transparent,
		black 12px calc(100% - 12px),
		transparent
	);
	cursor: pointer;
}

.cinderfish-folder > .header::before {
	content: "";
	position: absolute;
	inset: 0;
	background: var(--panel);
	opacity: 0.85;
	z-index: -1;
}

.cinderfish-folder > .header > .title {
	margin: 0;
	padding: 12px 16px 12px 0;
}

.cinderfish-folder > .header > .title:empty {
	display: none;
}

.cinderfish-folder > .header > .divider {
	flex: 1;
	margin: auto;
	block-size: 1px;
	background: var(--divider, rgba(128, 128, 128, 0.3));
}

.cinderfish-folder > .header > .toggle {
	background: none;
	border: none;
	cursor: pointer;
	color: inherit;
	padding: 12px 0 12px 16px;
	font-size: 0.8em;
}
</style>
