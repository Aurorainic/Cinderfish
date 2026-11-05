<!-- Ported from firefish@b425f61:packages/client/src/components/MkPoll.vue (AGPL-3.0-only) -->
<template>
	<div class="tivcixzd" :class="{ done: closed || isVoted }">
		<ul>
			<li
				v-for="(choice, i) in poll.choices"
				:key="i"
				:class="{ voted: choice.isVoted }"
				@click.stop="vote(i)"
			>
				<div
					class="backdrop"
					:style="{
						width: `${showResult ? (choice.votes / total) * 100 : 0}%`,
					}"
				></div>
				<span>
					<template v-if="choice.isVoted"
						><i class="ph-check ph-bold ph-lg"></i
					></template>
					<Mfm
						:text="choice.text"
						:plain="true"
						:custom-emojis="note.emojis"
					/>
					<span v-if="showResult" class="votes"
						>({{ t("_poll.votesCount", { n: choice.votes }) }})</span
					>
				</span>
			</li>
		</ul>
		<p v-if="!readOnly">
			<span>{{ t("_poll.totalVotes", { n: total }) }}</span>
			<span v-if="!closed && !isVoted">
				<span> · </span>
				<a @click.stop="showResult = !showResult">{{
					showResult ? t("_poll.vote") : t("_poll.showResult")
				}}</a>
			</span>
			<span v-if="isVoted"> · {{ t("_poll.voted") }}</span>
			<span v-else-if="closed"> · {{ t("_poll.closed") }}</span>
			<span v-if="remaining > 0"> · {{ timer }}</span>
		</p>
	</div>
</template>

<script lang="ts" setup>
// Degradations vs Firefish:
//   * `ap/show` does not exist, so a remote poll cannot be re-fetched ("reload"
//     link dropped);
//   * Cinderfish has no poll refresh endpoint either, so after voting the local
//     poll is updated optimistically instead of waiting for an update event.
import { computed, ref } from "vue";
import type { Note, NoteDetailed, Poll } from "@/api/types";
import { sum } from "@/scripts/array";
import { pleaseLogin } from "@/scripts/please-login";
import { useInterval } from "@/composables/use-interval";
import * as os from "@/os";
import { t } from "@/i18n";

const props = defineProps<{
	note: Note & Partial<Pick<NoteDetailed, "renote" | "reply" | "poll">>;
	readOnly?: boolean;
}>();

const remaining = ref(-1);

const emptyPoll: Poll = {
	multiple: false,
	expiresAt: null,
	choices: [],
	votesCount: 0,
};

const poll = computed<Poll>(() => props.note.poll ?? emptyPoll);
const total = computed(() => sum(poll.value.choices.map((x) => x.votes)));
const closed = computed(() => remaining.value === 0);
const isVoted = computed(
	() =>
		!poll.value.multiple && poll.value.choices.some((c) => c.isVoted),
);
const timer = computed(() =>
	t(
		remaining.value >= 86400
			? "_poll.remainingDays"
			: remaining.value >= 3600
			? "_poll.remainingHours"
			: remaining.value >= 60
			? "_poll.remainingMinutes"
			: "_poll.remainingSeconds",
		{
			s: Math.floor(remaining.value % 60),
			m: Math.floor(remaining.value / 60) % 60,
			h: Math.floor(remaining.value / 3600) % 24,
			d: Math.floor(remaining.value / 86400),
		},
	),
);

const showResult = ref(props.readOnly === true || isVoted.value);

// Poll with deadline
if (poll.value.expiresAt != null) {
	const expiresAt = poll.value.expiresAt;
	const tick = () => {
		remaining.value = Math.floor(
			Math.max(expiresAt - Date.now(), 0) / 1000,
		);
		if (remaining.value === 0) {
			showResult.value = true;
		}
	};

	useInterval(tick, 3000, {
		immediate: true,
		afterMounted: false,
	});
}

const vote = async (id: number): Promise<void> => {
	pleaseLogin();

	if (props.readOnly || closed.value || isVoted.value) return;

	const { canceled } = await os.confirm({
		type: "question",
		text: t("voteConfirm", {
			choice: poll.value.choices[id].text,
		}),
	});
	if (canceled) return;

	await os.apiWithDialog("notes/polls/vote", {
		noteId: props.note.id,
		choice: id,
	});

	// Optimistic local update: the vote endpoint returns `{}` and the poll is not
	// pushed over streaming.
	const choice = poll.value.choices[id];
	choice.isVoted = true;
	choice.votes += 1;
	poll.value.votesCount += 1;
	if (!showResult.value) showResult.value = !poll.value.multiple;
};
</script>

<style lang="scss" scoped>
.tivcixzd {
	> ul {
		display: block;
		margin: 0;
		padding: 0;
		list-style: none;

		> li {
			display: block;
			position: relative;
			margin: 4px 0;
			padding: 4px;
			//border: solid 0.5px var(--divider);
			background: var(--accentedBg);
			border-radius: 4px;
			overflow: hidden;
			cursor: pointer;

			> .backdrop {
				position: absolute;
				top: 0;
				left: 0;
				height: 100%;
				background: var(--accent);
				background: linear-gradient(
					90deg,
					var(--buttonGradateA),
					var(--buttonGradateB)
				);
				transition: width 1s ease;
			}

			> span {
				position: relative;
				display: inline-block;
				padding: 3px 5px;
				background: var(--panel);
				border-radius: 3px;

				> i {
					margin-right: 4px;
					color: var(--accent);
				}

				> .votes {
					margin-left: 4px;
					opacity: 0.7;
				}
			}
		}
	}

	> p {
		color: var(--fg);

		a {
			color: inherit;
		}
	}

	&.done {
		> ul > li {
			cursor: default;
		}
	}
}
</style>
