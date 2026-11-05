<!-- Ported from firefish@b425f61:packages/client/src/components/MkPollEditor.vue (AGPL-3.0-only) -->
<template>
	<div class="zmdxowus">
		<p v-if="choices.length < 2" class="caution">
			<i class="ph-warning ph-bold ph-lg"></i
			>{{ t("_poll.noOnlyOneChoice") }}
		</p>
		<ul>
			<li v-for="(choice, i) in choices" :key="i">
				<CfInput
					class="input"
					small
					:model-value="choice"
					:placeholder="t('_poll.choiceN', { n: i + 1 })"
					@update:modelValue="onInput(i, $event)"
				>
				</CfInput>
				<button
					class="_button"
					@click="remove(i)"
					:aria-label="t('remove')"
				>
					<i class="ph-x ph-bold ph-lg"></i>
				</button>
			</li>
		</ul>
		<CfButton v-if="choices.length < 10" class="add" @click="add">{{
			t("add")
		}}</CfButton>
		<CfButton v-else class="add" disabled>{{
			t("_poll.noMore")
		}}</CfButton>
		<CfSwitch v-model="multiple">{{
			t("_poll.canMultipleVote")
		}}</CfSwitch>
		<section>
			<div>
				<CfSelect v-model="expiration" small>
					<template #label>{{ t("_poll.expiration") }}</template>
					<option value="infinite">
						{{ t("_poll.infinite") }}
					</option>
					<option value="at">{{ t("_poll.at") }}</option>
					<option value="after">{{ t("_poll.after") }}</option>
				</CfSelect>
				<section v-if="expiration === 'at'">
					<CfInput v-model="atDate" small type="date" class="input">
						<template #label>{{
							t("_poll.deadlineDate")
						}}</template>
					</CfInput>
					<CfInput v-model="atTime" small type="time" class="input">
						<template #label>{{
							t("_poll.deadlineTime")
						}}</template>
					</CfInput>
				</section>
				<section v-else-if="expiration === 'after'">
					<CfInput v-model="after" small type="number" class="input">
						<template #label>{{ t("_poll.duration") }}</template>
					</CfInput>
					<CfSelect v-model="unit" small>
						<option value="second">
							{{ t("_time.second") }}
						</option>
						<option value="minute">
							{{ t("_time.minute") }}
						</option>
						<option value="hour">{{ t("_time.hour") }}</option>
						<option value="day">{{ t("_time.day") }}</option>
					</CfSelect>
				</section>
			</div>
		</section>
	</div>
</template>

<script lang="ts" setup>
// Degradation: Firefish seeds the deadline with `addTime()` from `scripts/time`
// and formats it with `formatDateTimeString()`; this file computes tomorrow's
// date directly so the editor stays self-contained.
import { ref, watch } from "vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfSelect from "@/components/ui/CfSelect.vue";
import CfSwitch from "@/components/ui/CfSwitch.vue";
import CfButton from "@/components/ui/CfButton.vue";
import { t } from "@/i18n";

interface PollEditorValue {
	choices: string[];
	multiple: boolean;
	expiresAt?: number;
	expiredAfter?: number;
}

const props = defineProps<{
	modelValue: PollEditorValue;
}>();

const emit = defineEmits<{
	(ev: "update:modelValue", v: PollEditorValue): void;
}>();

function toDateInputValue(date: Date): string {
	const y = date.getFullYear();
	const m = String(date.getMonth() + 1).padStart(2, "0");
	const d = String(date.getDate()).padStart(2, "0");
	return `${y}-${m}-${d}`;
}

const choices = ref(props.modelValue.choices);
const multiple = ref(props.modelValue.multiple);
const expiration = ref<"infinite" | "at" | "after">("infinite");
const atDate = ref(toDateInputValue(new Date(Date.now() + 86400000)));
const atTime = ref("00:00");
const after = ref(0);
const unit = ref<"second" | "minute" | "hour" | "day">("second");

if (props.modelValue.expiresAt != null) {
	expiration.value = "at";
	const date = new Date(props.modelValue.expiresAt);
	atDate.value = toDateInputValue(date);
	atTime.value = `${String(date.getHours()).padStart(2, "0")}:${String(
		date.getMinutes(),
	).padStart(2, "0")}`;
} else if (typeof props.modelValue.expiredAfter === "number") {
	expiration.value = "after";
	after.value = props.modelValue.expiredAfter / 1000;
} else {
	expiration.value = "infinite";
}

function onInput(i: number, value: string | number | null): void {
	choices.value[i] = value == null ? "" : String(value);
}

function add(): void {
	choices.value.push("");
}

function remove(i: number): void {
	choices.value = choices.value.filter((_, _i) => _i !== i);
}

function get(): PollEditorValue {
	const calcAt = (): number => {
		return new Date(`${atDate.value} ${atTime.value}`).getTime();
	};

	// Firefish walks the unit with switch fall-through (day -> hour -> minute ->
	// second); the factor table keeps the same result without disabling
	// `noFallthroughCasesInSwitch`.
	const calcAfter = (): number => {
		const base = parseInt(String(after.value));
		const factor =
			unit.value === "day"
				? 86400
				: unit.value === "hour"
				? 3600
				: unit.value === "minute"
				? 60
				: 1;
		return base * factor * 1000;
	};

	const result: PollEditorValue = {
		choices: choices.value,
		multiple: multiple.value,
	};
	if (expiration.value === "at") {
		result.expiresAt = calcAt();
	} else if (expiration.value === "after") {
		result.expiredAfter = calcAfter();
	}
	return result;
}

watch(
	[choices, multiple, expiration, atDate, atTime, after, unit],
	() => emit("update:modelValue", get()),
	{
		deep: true,
	},
);
</script>

<style lang="scss" scoped>
.zmdxowus {
	padding: 8px 16px;

	> .caution {
		margin: 0 0 8px 0;
		font-size: 0.8em;
		color: #f00;

		> i {
			margin-right: 4px;
		}
	}

	> ul {
		display: block;
		margin: 0;
		padding: 0;
		list-style: none;

		> li {
			display: flex;
			margin: 8px 0;
			padding: 0;
			width: 100%;

			> .input {
				flex: 1;
			}

			> button {
				width: 32px;
				padding: 4px 0;
			}
		}
	}

	> .add {
		margin: 8px 0;
		z-index: 1;
	}

	> section {
		margin: 16px 0 0 0;

		> div {
			margin: 0 8px;
			display: flex;
			flex-direction: row;
			flex-wrap: wrap;
			gap: 12px;

			&:last-child {
				flex: 1 0 auto;

				> div {
					flex-grow: 1;
				}

				> section {
					// MAGIC: Prevent div above from growing unless wrapped to its own line
					flex-grow: 9999;
					align-items: end;
					display: flex;
					gap: 4px;

					> .input {
						flex: 1 1 auto;
					}
				}
			}
		}
	}
}
</style>
