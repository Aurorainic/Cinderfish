<!-- Ported from firefish@b425f61:packages/client/src/pages/user/index.vue (AGPL-3.0-only) -->
<template>
	<UserIndex :acct="acct" :page="page" />
</template>

<script lang="ts" setup>
// The route is `/@:acct/:page?`, so `acct` is "alice" locally and
// "alice@remote.example" for a federated account. Like Firefish, the page
// passes `Acct.parse(acct)` on to `users/show`, which resolves a remote
// handle through the federation resolver.
import { computed } from "vue";
import { useRoute } from "vue-router";
import { definePageMetadata } from "@/scripts/page-metadata";
import UserIndex from "./UserIndex.vue";

const route = useRoute();

const acct = computed<string>(() => {
	const value = route.params.acct;
	return typeof value === "string" ? value : "";
});

const page = computed<string | undefined>(() => {
	const value = route.params.page;
	return typeof value === "string" && value !== "" ? value : undefined;
});

// UserIndex provides its own metadata (user name, avatar, follow button).
definePageMetadata(null);
</script>
