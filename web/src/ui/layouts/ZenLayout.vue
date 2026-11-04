<template>
	<div class="cf-zen">
		<router-view />
	</div>
</template>

<script setup lang="ts">
document.documentElement.style.overflowBlock = "scroll";
</script>

<style scoped>
.cf-zen {
	min-block-size: 100dvh;
	box-sizing: border-box;
}
</style>
