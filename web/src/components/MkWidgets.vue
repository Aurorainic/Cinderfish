<!-- Ported from firefish@b425f61:packages/client/src/components/MkWidgets.vue (AGPL-3.0-only) -->
<template>
	<aside class="widgets" aria-label="小部件">
		<header v-if="editMode" class="editor">
			<CfSelect v-model="selected" class="select">
				<template #label>{{ t("selectWidget") }}</template>
				<option value="" disabled>—</option>
				<option v-for="name in widgetNames" :key="name" :value="name">
					{{ widgetTitle(name) }}
				</option>
			</CfSelect>
			<div class="row">
				<CfButton primary :disabled="selected === ''" @click="addSelected">
					<i class="ph-bold ph-plus"></i> {{ t("add") }}
				</CfButton>
				<CfButton @click="exitEdit">{{ t("close") }}</CfButton>
			</div>
		</header>

		<div v-if="list.length === 0 && !editMode" class="empty _panel">
			<p>还没有小部件</p>
			<CfButton @click="enterEdit">{{ t("editWidgets") }}</CfButton>
		</div>

		<div
			v-for="(widget, index) in list"
			:key="widget.id"
			class="widget"
			@contextmenu.stop="onContextmenu(widget, $event)"
		>
			<div v-if="editMode" class="tools">
				<button class="_button" :aria-label="t('edit')" @click="configWidget(widget.id)">
					<i class="ph-bold ph-gear-six"></i>
				</button>
				<button
					class="_button"
					:disabled="index === 0"
					aria-label="上移"
					@click="move(widget.id, -1)"
				>
					<i class="ph-bold ph-arrow-up"></i>
				</button>
				<button
					class="_button"
					:disabled="index === list.length - 1"
					aria-label="下移"
					@click="move(widget.id, 1)"
				>
					<i class="ph-bold ph-arrow-down"></i>
				</button>
				<button class="_button" :aria-label="t('remove')" @click="removeWidget(widget.id)">
					<i class="ph-bold ph-x"></i>
				</button>
			</div>
			<component
				:is="widgetComponent(widget.name)"
				v-if="widgetComponent(widget.name) != null"
				:ref="(el: unknown) => setWidgetRef(widget.id, el)"
				:widget="widget"
				@updateProps="
					(data: Record<string, unknown>) => onUpdateProps(widget.id, data)
				"
			/>
			<div v-else class="unknown">未知的小部件：{{ widget.name }}</div>
		</div>

		<button v-if="editMode" class="_textButton footer" @click="exitEdit">
			<i class="ph-bold ph-check"></i> {{ t("editWidgetsExit") }}
		</button>
		<!-- The empty-state panel already carries an edit button; the footer
		     link renders only once there is something to edit, so a fresh
		     widget column does not show the same action twice. -->
		<button
			v-else-if="list.length > 0"
			class="_textButton footer"
			@click="enterEdit"
		>
			<i class="ph-bold ph-pencil"></i> {{ t("editWidgets") }}
		</button>
	</aside>
</template>

<script setup lang="ts">
// Firefish's widgets column: the shell renders it, it owns the widget list and
// the edit toggle. Differences from upstream, all forced by the contract:
//   * `vue-draggable-plus` is not a dependency (contract §2), so ordering uses
//     up/down buttons instead of dragging;
//   * `MkAd`, `MkButton` and `MkSelect` are replaced by CfButton/CfSelect;
//   * widgets are not registered globally as `Mkw*` — the registry resolves a
//     name to a lazily loaded component (`@/widgets`).
import { computed, ref } from "vue";
import { CfButton, CfSelect } from "@/components/ui";
import { popupMenu, type MenuItem } from "@/os";
import { t } from "@/i18n";
import {
	moveWidget,
	removeWidget,
	addWidget,
	updateWidgetData,
	widgetList,
	widgetStore,
} from "@/store/widgets";
import type { Widget } from "@/store/widgets";
import {
	widgetComponent,
	widgets as widgetNames,
	widgetTitle,
	type WidgetComponentExpose,
} from "@/widgets";

const list = computed(() => widgetList());

const editMode = computed<boolean>({
	get: () => widgetStore.reactiveState.editMode.value,
	set: (value) => widgetStore.set("editMode", value),
});

const selected = ref("");

const widgetRefs = new Map<string, WidgetComponentExpose>();

function setWidgetRef(id: string, el: unknown): void {
	const exposed = el as WidgetComponentExpose | null;
	if (exposed == null || typeof exposed.configure !== "function") {
		widgetRefs.delete(id);
		return;
	}
	widgetRefs.set(id, exposed);
}

function enterEdit(): void {
	editMode.value = true;
}

function exitEdit(): void {
	editMode.value = false;
}

function addSelected(): void {
	if (selected.value === "") return;
	addWidget(selected.value);
	selected.value = "";
}

function configWidget(id: string): void {
	widgetRefs.get(id)?.configure();
}

function move(id: string, delta: number): void {
	moveWidget(id, delta);
}

function onUpdateProps(id: string, data: Record<string, unknown>): void {
	updateWidgetData(id, data);
}

/** Firefish's guard: do not hijack the menu inside links, media or a selection. */
function onContextmenu(widget: Widget, event: MouseEvent): void {
	const target = event.target as HTMLElement | null;
	if (target == null) return;
	if (target.closest("a") != null) return;
	if (["INPUT", "TEXTAREA", "IMG", "VIDEO", "CANVAS"].includes(target.tagName)) return;
	if (target.isContentEditable) return;
	if ((window.getSelection()?.toString() ?? "") !== "") return;

	const items: MenuItem[] = [
		{ type: "label", text: widgetTitle(widget.name) },
		{
			icon: "ph-bold ph-gear-six",
			text: t("settings"),
			action: () => {
				configWidget(widget.id);
			},
		},
	];
	void popupMenu(items, event);
}
</script>

<style lang="scss" scoped>
.widgets {
	display: flex;
	flex-direction: column;
	min-height: 100vh;
	padding: var(--margin) 0;
	box-sizing: border-box;

	> .editor {
		display: flex;
		flex-direction: column;
		gap: 8px;
		margin-bottom: var(--margin);

		> .row {
			display: flex;
			gap: 8px;
		}
	}

	> .empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 12px;
		padding: 24px 16px;

		> p {
			margin: 0;
			font-size: 0.9em;
			color: var(--fgTransparentWeak);
		}
	}

	> .widget {
		position: relative;
		margin-bottom: var(--margin);

		> .tools {
			display: flex;
			justify-content: flex-end;
			gap: 4px;
			margin-bottom: 4px;

			> button {
				padding: 2px 6px;
				border-radius: 4px;
				background: var(--buttonBg);
				color: var(--fg);

				&:disabled {
					opacity: 0.4;
				}
			}
		}

		> .unknown {
			padding: 16px;
			font-size: 0.9em;
			color: var(--fgTransparentWeak);
		}
	}

	> .footer {
		align-self: center;
		font-size: 0.9em;
	}
}
</style>
