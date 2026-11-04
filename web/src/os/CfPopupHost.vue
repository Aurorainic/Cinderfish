<template>
	<component
		:is="entry.component"
		v-for="entry in popups"
		:key="entry.id"
		v-bind="entry.props"
		v-on="entry.events"
	/>
</template>

<script lang="ts" setup>
import { popups } from "./state";
</script>
