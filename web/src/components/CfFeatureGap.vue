<!-- Shared body for pages whose upstream feature has no backend here. -->
<template>
	<div class="cf-feature-gap">
		<CfInfo warn class="_formBlock">
			<strong>{{ title }}</strong>
			<p class="summary">{{ summary }}</p>
		</CfInfo>

		<section v-if="available.length > 0" class="_formBlock">
			<h2 class="sectionTitle">这里已经有</h2>
			<ul class="list">
				<li v-for="item in available" :key="item.label" class="item ok">
					<i class="ph-check-circle ph-bold"></i>
					<div class="body">
						<span class="label">{{ item.label }}</span>
						<span v-if="item.note != null" class="note">{{ item.note }}</span>
					</div>
					<RouterLink v-if="item.to != null" class="link" :to="item.to">
						打开
					</RouterLink>
				</li>
			</ul>
		</section>

		<section v-if="missing.length > 0" class="_formBlock">
			<h2 class="sectionTitle">上游有、这里没有</h2>
			<ul class="list">
				<li v-for="item in missing" :key="item" class="item">
					<i class="ph-x-circle ph-bold"></i>
					<div class="body">
						<span class="label">{{ item }}</span>
					</div>
				</li>
			</ul>
			<CfInfo class="note">
				这些功能需要对应的服务端支持（端点、表结构或外部服务），当前没有实现。
				与其放置点了没反应的表单，这个页面直接说明现状。
			</CfInfo>
		</section>
	</div>
</template>

<script lang="ts" setup>
// Several Firefish pages describe features this backend does not have. Rendering
// their forms would create controls that silently do nothing, so those pages
// share this body: it states what exists, what does not, and links to the
// pages that do work. AdminUnavailable is the same idea for the admin surface.
import CfInfo from "@/components/ui/CfInfo.vue";

defineOptions({ name: "CfFeatureGap" });

interface GapItem {
	label: string;
	note?: string;
	to?: string;
}

withDefaults(
	defineProps<{
		title: string;
		summary: string;
		available?: GapItem[];
		missing?: string[];
	}>(),
	{ available: () => [], missing: () => [] },
);
</script>

<style lang="scss" scoped>
.cf-feature-gap {
	.sectionTitle {
		font-size: 1.05em;
		font-weight: 700;
		margin: 0 0 12px;
	}

	.summary {
		margin: 8px 0 0;
		line-height: 1.6;
	}

	.list {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.item {
		display: flex;
		align-items: flex-start;
		gap: 10px;
		padding: 10px 12px;
		border-radius: var(--radius);
		background: var(--panelHighlight);

		&.ok > i {
			color: var(--success);
		}

		> i {
			margin-top: 2px;
			color: var(--error);
		}

		.body {
			flex: 1;
			display: flex;
			flex-direction: column;
			gap: 2px;
		}

		.label {
			font-weight: 600;
		}

		.note {
			opacity: 0.8;
			font-size: 0.92em;
		}

		.link {
			color: var(--accent);
			white-space: nowrap;
		}
	}
}
</style>
