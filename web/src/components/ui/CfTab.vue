<!-- Ported from firefish@b425f61:packages/client/src/components/MkTab.vue (AGPL-3.0-only) -->
<template>
	<div
		class="pxhvhrfw"
		:class="{ chips: variant === 'chips', underline: variant === 'underline' }"
		role="tablist"
	>
		<button
			v-for="tab in tabs"
			:key="tab.value"
			class="_button"
			role="tab"
			:aria-selected="tab.value === modelValue ? 'true' : 'false'"
			@click="select(tab.value)"
		>
			<i v-if="tab.icon" :class="iconClass(tab.icon)"></i>
			<span v-if="tab.count != null" class="count">{{ tab.count }}</span>
			<span class="label">{{ tab.label }}</span>
		</button>
	</div>
</template>

<script lang="ts">
/**
 * Tab entry for CfTab. Firefish takes the entries as `<option value>` slot
 * VNodes; Vue 3's compiler folds a native `<option>` into a single node and
 * drops its `value` prop, which is why the labels disappeared. The array
 * contract below replaces that slot (wave-3 contract §3.3).
 */
export interface CfTabItem {
	value: string | number;
	label: string;
	count?: number;
	icon?: string;
}
</script>

<script setup lang="ts">
import { computed } from "vue";

const props = defineProps<{
	modelValue: string | number;
	tabs: CfTabItem[];
	style?: string | Record<string, unknown>;
}>();

const emit = defineEmits<{
	"update:modelValue": [value: string | number];
}>();

// `style` is the Firefish prop name. A bound `:style="'chips'"` arrives as the
// string; a static `style="underline"` is parsed by the template compiler into a
// CSS declaration object, and a bare word has no `:`, so it arrives as `{}`.
// The contract documents the static `underline` form, so that empty object is
// read as `underline` instead of silently rendering the default pill style. A
// non-empty object is a layout declaration (`style="margin-bottom: ..."`), which
// means the caller asked for no variant.
const variant = computed<"chips" | "underline" | null>(() => {
	const raw = props.style;
	if (typeof raw === "string") {
		return raw === "chips" || raw === "underline" ? raw : null;
	}
	if (raw != null && typeof raw === "object") {
		return Object.keys(raw).length === 0 ? "underline" : null;
	}
	return null;
});

const WEIGHTS = new Set(["ph-bold", "ph-fill", "ph-duotone", "ph-light", "ph-thin"]);

// `icon` is a Phosphor glyph name (`chat`) or a full class (`ph-chat`); the
// weight class is added when the caller does not pick one.
function iconClass(name: string): string {
	const parts = name.split(/\s+/).filter(Boolean);
	const glyph = parts.some((part) => part.startsWith("ph-") && !WEIGHTS.has(part))
		? parts
		: [`ph-${parts[0]}`, ...parts.slice(1)];
	return (glyph.some((part) => WEIGHTS.has(part)) ? glyph : ["ph-bold", ...glyph]).join(" ");
}

function select(value: string | number): void {
	emit("update:modelValue", value);
}
</script>

<style lang="scss" scoped>
.pxhvhrfw {
	display: flex;
	font-size: 90%;
	border-radius: var(--radius);
	padding: 10px 8px;

	> button {
		flex: 1;
		padding: 10px 8px;
		margin: 0 8px;
		border-radius: var(--radius);
		transition:
			color 0.2s,
			background 0.2s;

		&[aria-selected="true"] {
			color: var(--accent);
			background: var(--accentedBg);
		}

		&:not([aria-selected="true"]):hover {
			color: var(--fgHighlighted);
			background: var(--panelHighlight);
		}

		&:disabled {
			opacity: 1;
			cursor: default;
		}
	}

	&.chips,
	&.underline {
		padding: 12px 90px 12px 32px;
		font-size: 0.85em;
		overflow-x: auto;
		white-space: nowrap;
		mask: linear-gradient(to right, black calc(100% - 90px), transparent);
		-webkit-mask: linear-gradient(to right, black calc(100% - 90px), transparent);

		&::-webkit-scrollbar {
			display: none;
		}

		> button {
			display: flex;
			align-items: center;
			gap: 6px;
			flex: unset;
			margin: 0 8px 0 0;
			padding: 0.5em 1em;
			border-radius: 100px;
			background: var(--buttonBg);

			> i {
				margin-top: -0.1em;
			}

			> .count {
				margin-right: -0.2em;
				font-variant-numeric: tabular-nums;
			}
		}
	}

	&.underline {
		padding-block: 0;
		margin-bottom: -1px;
		border-radius: 0;

		> button {
			background: none;
			border-radius: 0;
			padding-block: 10px;
			border-bottom: 2px solid transparent;

			&[aria-selected="true"] {
				background: none;
				font-weight: 700;
				border-bottom-color: var(--accent);
			}
		}
	}
}
</style>
