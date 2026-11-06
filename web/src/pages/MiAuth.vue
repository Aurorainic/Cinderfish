<!-- Ported from firefish@b425f61:packages/client/src/pages/miauth.vue (AGPL-3.0-only) -->
<template>
	<MkSpacer :content-max="800" :margin-min="16" :margin-max="32">
		<CfInfo v-if="session === ''" warn>
			MiAuth 链接缺少 <code>session</code> 参数，无法继续。请从应用重新发起授权。
		</CfInfo>
		<CfMiauth
			v-else
			:session="session"
			:name="name"
			:icon="icon"
			:permission="permission"
			:callback="callback"
		/>
	</MkSpacer>
</template>

<script lang="ts" setup>
// The approval UI already lives in the account layer (`CfMiauth`): it handles
// sign-in, permission display, the gen-token call and the callback redirect.
// This page only reads the query parameters Firefish's MiAuth link carries and
// rejects a link without a session.
import { computed } from "vue";
import { useRoute } from "vue-router";
import CfMiauth from "@/account/components/CfMiauth.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageMiAuth" });

const route = useRoute();

function first(value: unknown): string | undefined {
	if (typeof value === "string") return value;
	if (Array.isArray(value) && typeof value[0] === "string") return value[0];
	return undefined;
}

const session = computed(() => {
	const raw = route.params.session;
	return typeof raw === "string" ? raw : "";
});
const name = computed(() => first(route.query.name));
const icon = computed(() => first(route.query.icon));
const permission = computed(() => first(route.query.permission) ?? "");
const callback = computed(() => first(route.query.callback));

definePageMetadata({
	title: "第三方授权",
	icon: "ph-plugs-connected ph-bold ph-lg",
});
</script>
