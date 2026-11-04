<template>
	<div class="cf-page-loading"><span class="spinner" /></div>
</template>

<style scoped>
.cf-page-loading {
	display: flex;
	align-items: center;
	justify-content: center;
	min-block-size: 40vh;
}
.spinner {
	inline-size: 28px;
	block-size: 28px;
	border: solid 3px var(--accent);
	border-top-color: transparent;
	border-radius: 50%;
	animation: cf-page-spin 1s linear infinite;
}
@keyframes cf-page-spin {
	to { transform: rotate(360deg); }
}
</style>
