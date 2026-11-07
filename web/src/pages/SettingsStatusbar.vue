<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/statusbar.vue (AGPL-3.0-only) -->
<template>
	<div class="_formRoot cf-settings-statusbar">
		<CfInfo class="_formBlock">
			状态栏固定在页面底部，可以放多条。每条独立设置高度、名称与深色底。
			目前只有「联邦」一种类型：它滚动显示已收录的远端实例主机名。
		</CfInfo>

		<CfInfo v-if="items.length === 0" class="_formBlock">
			还没有任何条目。添加一条后它马上出现在页面底部。
		</CfInfo>

		<section
			v-for="(item, index) in items"
			:key="item.id"
			class="card _formBlock"
		>
			<header class="head">
				<h2 class="title">{{ item.name || "（未命名）" }}</h2>
				<div class="headActions">
					<CfButton
						mini
						:disabled="index === 0"
						title="上移"
						@click="move(index, -1)"
					>
						<i class="ph-arrow-up ph-bold"></i>
					</CfButton>
					<CfButton
						mini
						:disabled="index === items.length - 1"
						title="下移"
						@click="move(index, 1)"
					>
						<i class="ph-arrow-down ph-bold"></i>
					</CfButton>
					<CfButton mini danger @click="remove(item.id)">
						<i class="ph-trash ph-bold"></i>
					</CfButton>
				</div>
			</header>

			<CfInput
				:model-value="item.name"
				class="field"
				@update:model-value="patch(item.id, { name: String($event ?? '') })"
			>
				<template #label>名称</template>
			</CfInput>

			<CfSelect
				:model-value="item.type"
				class="field"
				@update:model-value="patch(item.id, { type: String($event ?? 'federation') })"
			>
				<template #label>类型</template>
				<option value="federation">联邦实例</option>
				<option value="rss">RSS</option>
				<option value="" disabled>用户列表（首版不做列表）</option>
			</CfSelect>

			<CfSelect
				:model-value="item.size"
				class="field"
				@update:model-value="onSize(item.id, $event)"
			>
				<template #label>高度</template>
				<option value="verySmall">极小</option>
				<option value="small">小</option>
				<option value="medium">中</option>
				<option value="large">大</option>
				<option value="veryLarge">极大</option>
			</CfSelect>

			<CfSwitch
				:model-value="item.black"
				class="field"
				@update:model-value="patch(item.id, { black: $event === true })"
			>
				深色背景
			</CfSwitch>

			<template v-if="item.type === 'federation'">
				<CfSwitch
					:model-value="item.props.colored === true"
					class="field"
					@update:model-value="patchProps(item.id, { colored: $event === true })"
				>
					为每条着色
					<template #caption>
						用实例主机名派生一个稳定色相做底纹（接口不提供主题色）。
					</template>
				</CfSwitch>
				<CfSwitch
					:model-value="item.props.marqueeReverse === true"
					class="field"
					@update:model-value="
						patchProps(item.id, { marqueeReverse: $event === true })
					"
				>
					反向滚动
				</CfSwitch>
				<CfInput
					:model-value="String(item.props.marqueeDuration ?? 40)"
					class="field"
					type="number"
					@update:model-value="
						patchProps(item.id, { marqueeDuration: clampNumber($event, 5, 600, 40) })
					"
				>
					<template #label>滚动一轮秒数</template>
				</CfInput>
				<CfInput
					:model-value="String(item.props.refreshIntervalSec ?? 300)"
					class="field"
					type="number"
					@update:model-value="
						patchProps(item.id, {
							refreshIntervalSec: clampNumber($event, 30, 3600, 300),
						})
					"
				>
					<template #label>刷新间隔（秒）</template>
					<template #caption>最短 30 秒；列表来自本地库，不会请求远端。</template>
				</CfInput>
			</template>

			<template v-if="item.type === 'rss'">
				<CfInput
					:model-value="String(item.props.url ?? '')"
					class="field"
					@update:model-value="patchProps(item.id, { url: String($event ?? '').trim() })"
				>
					<template #label>订阅地址</template>
					<template #caption>
						由服务端抓取（<code>fetch-rss</code>），浏览器不会跨域请求；
						地址必须是 http(s)，指向内网或回环地址会被拒绝。
					</template>
				</CfInput>
				<CfSwitch
					:model-value="item.props.shuffle === true"
					class="field"
					@update:model-value="patchProps(item.id, { shuffle: $event === true })"
				>
					打乱顺序
				</CfSwitch>
				<CfSwitch
					:model-value="item.props.marqueeReverse === true"
					class="field"
					@update:model-value="
						patchProps(item.id, { marqueeReverse: $event === true })
					"
				>
					反向滚动
				</CfSwitch>
				<CfInput
					:model-value="String(item.props.marqueeDuration ?? 40)"
					class="field"
					type="number"
					@update:model-value="
						patchProps(item.id, { marqueeDuration: clampNumber($event, 5, 600, 40) })
					"
				>
					<template #label>滚动一轮秒数</template>
				</CfInput>
				<CfInput
					:model-value="String(item.props.refreshIntervalSec ?? 300)"
					class="field"
					type="number"
					@update:model-value="
						patchProps(item.id, {
							refreshIntervalSec: clampNumber($event, 30, 3600, 300),
						})
					"
				>
					<template #label>刷新间隔（秒）</template>
					<template #caption>最短 30 秒；每次刷新都会由服务端重新抓取订阅源。</template>
				</CfInput>
			</template>
		</section>

		<div class="actions _formBlock">
			<CfButton primary @click="add">
				<i class="ph-plus ph-bold ph-lg"></i>
				添加条目
			</CfButton>
			<CfButton v-if="items.length > 0" danger @click="clearAll">
				全部清除
			</CfButton>
		</div>
	</div>
</template>

<script lang="ts" setup>
// Firefish's statusbar settings page. The port keeps add / remove / reorder /
// size / name / black-mode and the federation item's own knobs, and drops the
// types this instance cannot serve (rss, userList) instead of rendering them.
// Every edit writes straight to the store, so the bar at the bottom of the
// layout updates without a save button.
import { computed } from "vue";
import type { StatusbarItem } from "@/store";
import { defaultStore } from "@/store";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfSelect from "@/components/ui/CfSelect.vue";
import CfSwitch from "@/components/ui/CfSwitch.vue";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageSettingsStatusbar" });

const items = computed<StatusbarItem[]>(
	() => defaultStore.reactiveState.statusbars.value,
);

function newID(): string {
	if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
		return crypto.randomUUID();
	}
	return `sb-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

function add(): void {
	defaultStore.push("statusbars", {
		id: newID(),
		name: "",
		type: "federation",
		size: "medium",
		black: false,
		props: {},
	});
}

function remove(id: string): void {
	defaultStore.set(
		"statusbars",
		items.value.filter((item) => item.id !== id),
	);
}

function clearAll(): void {
	defaultStore.set("statusbars", []);
}

function move(index: number, delta: number): void {
	const next = [...items.value];
	const target = index + delta;
	if (target < 0 || target >= next.length) return;
	const [row] = next.splice(index, 1);
	next.splice(target, 0, row);
	defaultStore.set("statusbars", next);
}

function patch(id: string, update: Partial<StatusbarItem>): void {
	defaultStore.set(
		"statusbars",
		items.value.map((item) => (item.id === id ? { ...item, ...update } : item)),
	);
}

const sizes: StatusbarItem["size"][] = [
	"verySmall",
	"small",
	"medium",
	"large",
	"veryLarge",
];

function onSize(id: string, value: unknown): void {
	const size = sizes.find((s) => s === value) ?? "medium";
	patch(id, { size });
}

function patchProps(id: string, update: Record<string, unknown>): void {
	defaultStore.set(
		"statusbars",
		items.value.map((item) =>
			item.id === id ? { ...item, props: { ...item.props, ...update } } : item,
		),
	);
}

function clampNumber(
	value: unknown,
	min: number,
	max: number,
	fallback: number,
): number {
	const parsed = Number(value);
	if (!Number.isFinite(parsed)) return fallback;
	return Math.min(max, Math.max(min, Math.round(parsed)));
}

definePageMetadata({ title: "状态栏", icon: "ph-list-bullets ph-bold ph-lg" });
</script>

<style lang="scss" scoped>
.cf-settings-statusbar {
	.card {
		padding: 14px 16px;
		border-radius: var(--radius);
		background: var(--panel);
	}

	.head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		margin-bottom: 12px;

		.title {
			margin: 0;
			font-size: 1em;
			overflow: hidden;
			text-overflow: ellipsis;
			white-space: nowrap;
		}

		.headActions {
			display: flex;
			gap: 6px;
			flex-shrink: 0;
		}
	}

	.field {
		margin-bottom: 10px;
	}

	.actions {
		display: flex;
		gap: 8px;
		flex-wrap: wrap;
	}
}
</style>
