<template>
	<div class="cf-page-placeholder">
		<p>TODO: port pages/_error_.vue</p>
		<button class="cf-page-retry" @click="$router.go(0)">retry</button>
	</div>
</template>

<style scoped>
.cf-page-placeholder {
	padding: 48px 24px;
	text-align: center;
	color: var(--fgTransparentWeak);
}
.cf-page-retry {
	margin-block-start: 12px;
	padding: 6px 16px;
	color: var(--fg);
	background: var(--buttonBg);
	border: none;
	border-radius: 5px;
	cursor: pointer;
}
</style>
