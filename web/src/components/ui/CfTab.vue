<script lang="ts">
import { defineComponent, h, type PropType } from "vue";

type TabValue = string | number;

/**
 * Port of Firefish `components/MkTab.vue`.
 *
 * Firefish uses a render function because the options are arbitrary slot
 * VNodes (`<button value="...">`). Keeping the render function preserves that
 * call shape exactly.
 */
export default defineComponent({
	props: {
		modelValue: {
			type: [String, Number] as PropType<TabValue>,
			required: true,
		},
		style: {
			type: String as PropType<"chips" | "underline" | undefined>,
			default: undefined,
		},
	},
	emits: ["update:modelValue"],
	setup(props, { slots, emit }) {
		return () => {
			const options = slots.default?.() ?? [];
			return h(
				"div",
				{
					class: [
						"pxhvhrfw",
						{ chips: props.style === "chips" },
						{ underline: props.style === "underline" },
					],
					role: "tablist",
				},
				options.map((option) => {
					const optionProps = (option.props ?? {}) as {
						value?: TabValue;
					};
					return h(
						"button",
						{
							class: "_button",
							role: "tab",
							key: option.key as string | number | symbol,
							"aria-selected":
								props.modelValue === optionProps.value ? "true" : "false",
							onClick: () => {
								if (optionProps.value !== undefined) {
									emit("update:modelValue", optionProps.value);
								}
							},
						},
						option.children ?? undefined,
					);
				}),
			);
		};
	},
});
</script>

<style lang="scss">
.pxhvhrfw {
	display: flex;
	font-size: 90%;
	border-radius: var(--radius);
	padding-block: 10px;
	padding-inline: 8px;
	gap: 4px;

	> button {
		flex: 1;
		padding-block: 10px;
		padding-inline: 12px;
		border-radius: 6px;
		color: var(--fgTransparentWeak);
		font-weight: bold;
		transition:
			color 0.2s,
			background 0.2s;

		&:hover {
			background: var(--buttonBg);
			color: var(--fg);
		}

		&[aria-selected="true"] {
			color: var(--accent);
			background: var(--accentedBg);
		}
	}

	&.chips > button {
		border: solid 1px var(--divider);
		border-radius: 999px;
		font-size: 0.9em;
	}

	&.underline {
		gap: 0;
		padding-inline: 0;
		border-radius: 0;
		border-block-end: solid 1px var(--divider);

		> button {
			border-radius: 0;
			background: none;
			border-block-end: solid 2px transparent;
		}

		> button[aria-selected="true"] {
			background: none;
			border-block-end-color: var(--accent);
		}
	}
}
</style>
