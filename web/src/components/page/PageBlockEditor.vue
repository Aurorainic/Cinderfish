<!-- Ported from firefish@b425f61:packages/client/src/pages/page-editor/page-editor.blocks.vue (AGPL-3.0-only) -->
<template>
	<div class="cf-page-block-editor" :class="{ nested: depth > 1 }">
		<div
			v-for="(block, index) in blocks"
			:key="blockKey(block, index)"
			class="block _panel"
		>
			<div class="head">
				<span class="type">{{ blockLabel(block.type) }}</span>
				<span class="actions">
					<button
						v-tooltip="'Move up'"
						class="_button action"
						:disabled="index === 0"
						@click="move(index, -1)"
					>
						<i class="ph-arrow-up ph-bold"></i>
					</button>
					<button
						v-tooltip="'Move down'"
						class="_button action"
						:disabled="index === blocks.length - 1"
						@click="move(index, 1)"
					>
						<i class="ph-arrow-down ph-bold"></i>
					</button>
					<button
						v-tooltip="t('remove')"
						class="_button action remove"
						@click="removeAt(index)"
					>
						<i class="ph-trash ph-bold"></i>
					</button>
				</span>
			</div>

			<div class="body">
				<CfTextarea
					v-if="block.type === 'text'"
					:model-value="stringField(block, 'text')"
					@update:model-value="setStringField(block, 'text', $event)"
				>
					<template #label>Text (MFM)</template>
				</CfTextarea>

				<template v-else-if="block.type === 'section'">
					<CfInput
						:model-value="stringField(block, 'title')"
						@update:model-value="setStringField(block, 'title', $event)"
					>
						<template #label>Section title</template>
					</CfInput>
					<div class="children">
						<PageBlockEditor
							:blocks="childrenOf(block)"
							:attached-files="attachedFiles"
							:depth="depth + 1"
							@file-picked="(file) => emit('file-picked', file)"
						/>
					</div>
				</template>

				<template v-else-if="block.type === 'image'">
					<div class="image">
						<MkImgWithBlurhash
							v-if="fileFor(block) != null"
							:src="resolveMediaUrl(fileFor(block)?.url)"
							:hash="fileFor(block)?.blurhash"
							:alt="fileFor(block)?.name"
							:cover="false"
						/>
						<span v-else class="empty">No file selected</span>
					</div>
					<div class="row">
						<CfButton @click="pickImage(block)">
							<i class="ph-image-square ph-bold ph-lg"></i>
							{{ t("attachFile") }}
						</CfButton>
						<span v-if="fileFor(block) != null" class="fileName">
							{{ fileFor(block)?.name }}
						</span>
					</div>
				</template>

				<template v-else-if="block.type === 'note'">
					<CfInput
						:model-value="stringField(block, 'note')"
						@update:model-value="setStringField(block, 'note', $event)"
					>
						<template #label>Note ID</template>
						<template #caption>
							The id after <code>/notes/</code>. The viewer fetches the
							note through <code>notes/show</code>.
						</template>
					</CfInput>
				</template>

				<div v-else class="unsupported">
					<i class="ph-puzzle-piece ph-bold ph-lg"></i>
					<span>
						This <strong>{{ block.type }}</strong> block is preserved but
						not rendered by Cinderfish. It can only be removed here.
					</span>
				</div>
			</div>
		</div>

		<div class="add" :class="{ empty: blocks.length === 0 }">
			<span class="label">{{ blocks.length === 0 ? "Add a block:" : "Add:" }}</span>
			<CfButton
				v-for="type in addableTypes"
				:key="type.value"
				mini
				@click="addBlock(type.value)"
			>
				<i :class="type.icon"></i>
				{{ type.label }}
			</CfButton>
		</div>
	</div>
</template>

<script setup lang="ts">
// The Firefish editor drags blocks with vue-draggable-plus; that dependency is
// outside the port contract, so ordering is expressed with explicit up/down
// buttons. Only the four statically rendered types can be added: upstream's
// own editor offers exactly those and always sends `script: ''` /
// `variables: []` (c05e844). A dynamic block already present in a
// Firefish-authored document stays in the array untouched and is shown as a
// removable placeholder.
import { computed } from "vue";
import { resolveMediaUrl } from "@/api/client";
import type { DriveFile, PageBlock, PageBlockType } from "@/api/types";
import MkImgWithBlurhash from "@/components/MkImgWithBlurhash.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfTextarea from "@/components/ui/CfTextarea.vue";
import { t } from "@/i18n";
import { selectFile } from "@/scripts/select-file";

defineOptions({ name: "PageBlockEditor" });

/** Keep the page shallow enough to stay readable on a phone. */
const MAX_DEPTH = 4;

const props = withDefaults(
	defineProps<{
		blocks: PageBlock[];
		attachedFiles?: DriveFile[];
		depth?: number;
	}>(),
	{
		attachedFiles: () => [],
		depth: 1,
	},
);

const emit = defineEmits<{
	"file-picked": [file: DriveFile];
}>();

interface AddableType {
	value: PageBlockType;
	label: string;
	icon: string;
}

const STATIC_TYPES: AddableType[] = [
	{ value: "text", label: "Text", icon: "ph-text-t ph-bold ph-lg" },
	{ value: "section", label: "Section", icon: "ph-rows ph-bold ph-lg" },
	{ value: "image", label: "Image", icon: "ph-image-square ph-bold ph-lg" },
	{ value: "note", label: "Note", icon: "ph-file-text ph-bold ph-lg" },
];

const addableTypes = computed<AddableType[]>(() =>
	props.depth >= MAX_DEPTH
		? STATIC_TYPES.filter((type) => type.value !== "section")
		: STATIC_TYPES,
);

const BLOCK_LABELS: Record<string, string> = {
	text: "Text",
	section: "Section",
	image: "Image",
	note: "Note",
};

function blockLabel(type: string): string {
	return BLOCK_LABELS[type] ?? type;
}

function blockKey(block: PageBlock, index: number): string {
	return typeof block.id === "string" ? block.id : `${props.depth}-${index}-${block.type}`;
}

function stringField(block: PageBlock, key: string): string {
	const value = block[key];
	return typeof value === "string" ? value : "";
}

function setStringField(
	block: PageBlock,
	key: string,
	value: string | number | null,
): void {
	block[key] = value == null ? "" : String(value);
}

function childrenOf(block: PageBlock): PageBlock[] {
	const value = block.children;
	return Array.isArray(value)
		? value.filter((child): child is PageBlock => child != null && typeof child === "object")
		: [];
}

function fileFor(block: PageBlock): DriveFile | null {
	const fileId = stringField(block, "fileId");
	if (fileId === "") return null;
	return props.attachedFiles.find((file) => file.id === fileId) ?? null;
}

function createBlock(type: PageBlockType): PageBlock {
	const id = crypto.randomUUID();
	switch (type) {
		case "section":
			return { id, type, title: "", children: [] };
		case "image":
			return { id, type, fileId: "" };
		case "note":
			return { id, type, note: "" };
		case "text":
		default:
			return { id, type: "text", text: "" };
	}
}

function addBlock(type: PageBlockType): void {
	props.blocks.push(createBlock(type));
}

function removeAt(index: number): void {
	props.blocks.splice(index, 1);
}

function move(index: number, delta: number): void {
	const next = index + delta;
	if (next < 0 || next >= props.blocks.length) return;
	const [block] = props.blocks.splice(index, 1);
	props.blocks.splice(next, 0, block);
}

async function pickImage(block: PageBlock): Promise<void> {
	const file = await selectFile({ label: t("attachFile") });
	if (file == null) return;
	block.fileId = file.id;
	emit("file-picked", file);
}
</script>

<style lang="scss" scoped>
.cf-page-block-editor {
	&.nested {
		margin-block-start: 12px;
		padding-inline-start: 12px;
		border-inline-start: solid 2px var(--divider);
	}

	> .block {
		padding: 12px;

		& + .block {
			margin-block-start: 12px;
		}

		> .head {
			display: flex;
			align-items: center;
			gap: 8px;
			margin-block-end: 8px;
			font-size: 0.9em;

			> .type {
				font-weight: bold;
				color: var(--fgTransparentWeak);
			}

			> .actions {
				display: inline-flex;
				align-items: center;
				gap: 2px;
				margin-inline-start: auto;

				> .action {
					padding: 4px 6px;
					color: var(--fgTransparentWeak);

					&:disabled {
						opacity: 0.35;
						cursor: default;
					}

					&.remove {
						color: var(--danger);
					}
				}
			}
		}
	}

	.image {
		display: flex;
		justify-content: center;
		margin-block-end: 8px;

		img {
			display: block;
			max-inline-size: 100%;
			max-block-size: 200px;
			object-fit: contain;
		}

		.empty {
			color: var(--fgTransparentWeak);
		}
	}

	.row {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 8px;

		.fileName {
			font-size: 0.9em;
			color: var(--fgTransparentWeak);
			overflow-wrap: anywhere;
		}
	}

	.children {
		margin-block-start: 12px;
	}

	.unsupported {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 12px;
		color: var(--fgTransparentWeak);
		font-size: 0.9em;
		background: var(--panelHighlight);
		border: dashed 1px var(--divider);
		border-radius: var(--radius);
	}

	> .add {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 8px;
		margin-block-start: 12px;

		&.empty {
			margin-block-start: 0;
		}

		> .label {
			font-size: 0.9em;
			color: var(--fgTransparentWeak);
		}
	}
}
</style>
