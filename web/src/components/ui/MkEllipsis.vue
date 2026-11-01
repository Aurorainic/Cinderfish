// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/global/MkEllipsis.vue
<template>
	<span class="cinderfish-ellipsis">
		<span>.</span><span>.</span><span>.</span>
	</span>
</template>

<style scoped>
.cinderfish-ellipsis > span {
	animation: ellipsis 1.4s infinite ease-in-out both;
}

.cinderfish-ellipsis > span:nth-child(1) {
	animation-delay: 0s;
}

.cinderfish-ellipsis > span:nth-child(2) {
	animation-delay: 0.16s;
}

.cinderfish-ellipsis > span:nth-child(3) {
	animation-delay: 0.32s;
}

@keyframes ellipsis {
	0%,
	80%,
	100% {
		opacity: 1;
	}
	40% {
		opacity: 0;
	}
}
</style>
