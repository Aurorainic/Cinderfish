<!-- Ported from firefish@b425f61:packages/client/src/components/global/MkError.vue (AGPL-3.0-only) -->
<template>
	<div class="mjndxjcg">
		<i class="ph-cloud-x ph-bold icon"></i>
		<p>
			<i class="ph-warning ph-bold ph-lg"></i>
			{{ t("somethingHappened") }}
		</p>
		<CfButton class="button" @click="emit('retry')">{{
			t("retry")
		}}</CfButton>
	</div>
</template>

<script setup lang="ts">
import CfButton from "../ui/CfButton.vue";
import { t } from "../../i18n";

// Degraded versus Firefish: the /static-assets/badges/error.png badge is not
// shipped, so the ghost image is replaced by a Phosphor glyph.
const emit = defineEmits<{
	retry: [];
}>();
</script>

<style lang="scss" scoped>
.mjndxjcg {
	padding: 32px;
	text-align: center;

	> p {
		margin: 0 0 8px 0;
	}

	> .button {
		margin: 0 auto;
	}

	> .icon {
		display: block;
		font-size: 64px;
		opacity: 0.5;
		margin: 0 auto 16px auto;
	}
}
</style>
