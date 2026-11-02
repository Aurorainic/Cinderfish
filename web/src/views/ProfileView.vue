// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/pages/user/index.vue
//
// The router hands the view a `handle` param; the original's `acct` prop is
// parsed in the same way (local part only — remote hosts are not resolvable
// by users/show, and UserHome renders the unavailable state for those).
<script lang="ts" setup>
import { computed } from "vue";
import { useRoute } from "vue-router";
import UserHome from "./profile/UserHome.vue";

const route = useRoute();

const username = computed(() => {
	const raw = route.params.handle;
	const value = Array.isArray(raw) ? (raw[0] ?? "") : String(raw ?? "");
	const at = value.indexOf("@");
	return at < 0 ? value : value.slice(0, at);
});
</script>

<template>
	<UserHome :username="username" />
</template>
