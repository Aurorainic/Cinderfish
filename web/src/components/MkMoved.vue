<!-- Ported from firefish@b425f61:packages/client/src/components/MkMoved.vue (AGPL-3.0-only) -->
<template>
	<div class="cf-moved">
		<i class="ph-airplane-takeoff ph-bold ph-lg icon"></i>
		{{ t("accountMoved") }}
		<MkMention class="link" :username="acct" :host="host" />
	</div>
</template>

<script setup lang="ts">
import MkMention from "@/components/MkMention.vue";
import { t } from "@/i18n";

defineOptions({ name: "MkMoved" });

defineProps<{
	acct: string;
	host: string;
}>();
</script>

<style lang="scss" scoped>
.cf-moved {
	padding: 16px;
	font-size: 90%;
	background: var(--infoWarnBg);
	color: var(--error);
	border-radius: var(--radius);

	> .icon {
		margin-right: 8px;
	}

	> .link {
		margin-left: 4px;
		color: var(--accent);
	}
}
</style>
