import type { ComputedRef, InjectionKey } from "vue";
export type RadioValue = string | number | boolean | null;

export interface RadiosContext {
	model: ComputedRef<RadioValue>;
	select: (value: RadioValue) => void;
	disabled: ComputedRef<boolean>;
}

export const RADIOS_KEY: InjectionKey<RadiosContext> = Symbol("cinderfish:radios");
