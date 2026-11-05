// Ported from firefish@b425f61:packages/client/src/widgets/index.ts (AGPL-3.0-only)
//
// Firefish registers every widget globally as `Mkw*` and keeps a `widgets`
// array of names for the picker. Cinderfish registers nothing globally: the
// widgets column resolves a name to a lazily loaded component through
// `widgetComponent()`, so a widget is only downloaded when it is on screen.
//
// First wave: post-form, notifications, timeline, trends, memo, instance-cloud,
// activity, photos, user-list, clock, digital-clock, server-info. Widgets whose
// data has no endpoint (rss, rss-ticker, aiscript, job-queue, server-metric,
// online-users, calendar/federation/unix-clock/slideshow/button) are not ported.
import { defineAsyncComponent, reactive, type Component } from "vue";
import { popup } from "@/os/state";
import { t } from "@/i18n";
import WidgetConfigDialog from "./WidgetConfigDialog.vue";
import type { Widget, WidgetData } from "@/store/widgets";

// ---------------------------------------------------------------------------
// Widget prop manager (Firefish's `widgets/widget.ts`)
// ---------------------------------------------------------------------------

export type WidgetFieldType = "boolean" | "number" | "text" | "enum";

export interface WidgetFieldOption {
	value: string | number | null;
	label: string;
}

export interface WidgetField {
	key: string;
	label: string;
	type: WidgetFieldType;
	default: unknown;
	min?: number;
	max?: number;
	step?: number;
	options?: WidgetFieldOption[];
}

export interface WidgetManager {
	/** Live widget props; the widget reads and mutates this object. */
	widgetProps: WidgetData;
	/** Persist the current props into `defaultStore.widgets`. */
	save: () => void;
	/** Open the config dialog (bound to the gear button and context menu). */
	configure: () => void;
}

export interface WidgetComponentExpose {
	name: string;
	id: string | null;
	configure: () => void;
}

function defaultsOf(fields: WidgetField[]): WidgetData {
	const values: WidgetData = {};
	for (const field of fields) values[field.key] = field.default;
	return values;
}

/**
 * Firefish's `useWidgetPropsManager`. The only deliberate differences:
 *   * `os.form` does not exist here, so the config UI is a small dialog
 *     (`WidgetConfigDialog.vue`) driven by the same field definition;
 *   * `throttle-debounce` is not a dependency (contract §2), so `save()` is a
 *     plain debounce.
 */
export function useWidgetPropsManager(
	name: string,
	fields: WidgetField[],
	widget: Widget | undefined,
	emit: (data: WidgetData) => void,
): WidgetManager {
	const widgetProps = reactive<WidgetData>({
		...defaultsOf(fields),
		...(widget?.data ?? {}),
	});

	let timer: number | null = null;

	const save = (): void => {
		if (timer != null) window.clearTimeout(timer);
		timer = window.setTimeout(() => {
			timer = null;
			emit({ ...widgetProps });
		}, 300);
	};

	const configure = (): void => {
		const currentDef = fields.map((field) => ({
			...field,
			default: widgetProps[field.key],
		}));
		popup(
			WidgetConfigDialog,
			{
				title: t(`_widgets.${name}`),
				fields: currentDef,
			},
			{
				done: (values: WidgetData) => {
					Object.assign(widgetProps, values);
					save();
				},
			},
			"closed",
		);
	};

	return { widgetProps, save, configure };
}

// ---------------------------------------------------------------------------
// Clock helpers shared by the two clock widgets
// ---------------------------------------------------------------------------

export interface ClockParts {
	hour: number;
	minute: number;
	second: number;
	millisecond: number;
	/** e.g. "+08:00" for the configured or local zone. */
	offsetLabel: string;
	/** e.g. "GMT+8" or "CST". */
	zoneLabel: string;
}

/** Real IANA zones when the engine exposes them, otherwise "(auto)" only. */
export function timezoneOptions(): WidgetFieldOption[] {
	const intl = Intl as unknown as {
		supportedValuesOf?: (key: string) => string[];
	};
	const zones = intl.supportedValuesOf?.("timeZone") ?? [];
	return [
		{ value: null, label: "(auto)" },
		...zones.map((zone) => ({ value: zone, label: zone })),
	];
}

function offsetMinutes(date: Date, timeZone: string | null): number {
	if (timeZone == null) return -date.getTimezoneOffset();
	const name = new Intl.DateTimeFormat("en-US", {
		timeZone,
		timeZoneName: "shortOffset",
	})
		.formatToParts(date)
		.find((part) => part.type === "timeZoneName")?.value;
	const match = name?.match(/GMT([+-])(\d{1,2})(?::(\d{2}))?/);
	if (match == null) return 0;
	const sign = match[1] === "-" ? -1 : 1;
	return sign * (Number(match[2]) * 60 + Number(match[3] ?? 0));
}

function formatOffset(minutes: number): string {
	const sign = minutes >= 0 ? "+" : "-";
	const absolute = Math.abs(minutes);
	const hours = Math.floor(absolute / 60)
		.toString()
		.padStart(2, "0");
	const rest = (absolute % 60).toString().padStart(2, "0");
	return `${sign}${hours}:${rest}`;
}

export function clockParts(date: Date, timeZone: string | null): ClockParts {
	const parts = new Intl.DateTimeFormat("en-GB", {
		hourCycle: "h23",
		hour: "2-digit",
		minute: "2-digit",
		second: "2-digit",
		...(timeZone != null ? { timeZone } : {}),
	}).formatToParts(date);
	const pick = (type: string): number =>
		Number(parts.find((part) => part.type === type)?.value ?? 0);

	const zoneName = new Intl.DateTimeFormat("en-US", {
		...(timeZone != null ? { timeZone } : {}),
		timeZoneName: "short",
	})
		.formatToParts(date)
		.find((part) => part.type === "timeZoneName")?.value;

	return {
		hour: pick("hour") % 24,
		minute: pick("minute"),
		second: pick("second"),
		millisecond: date.getMilliseconds(),
		offsetLabel: formatOffset(offsetMinutes(date, timeZone)),
		zoneLabel: zoneName ?? "?",
	};
}

// ---------------------------------------------------------------------------
// Registry
// ---------------------------------------------------------------------------

interface WidgetLoader {
	name: string;
	load: () => Promise<{ default: Component }>;
}

/** Picker order follows Firefish's `widgets` array, filtered to the 12 ports. */
const loaders: WidgetLoader[] = [
	{ name: "memo", load: () => import("./memo.vue") },
	{ name: "notifications", load: () => import("./notifications.vue") },
	{ name: "timeline", load: () => import("./timeline.vue") },
	{ name: "trends", load: () => import("./trends.vue") },
	{ name: "clock", load: () => import("./clock.vue") },
	{ name: "activity", load: () => import("./activity.vue") },
	{ name: "photos", load: () => import("./photos.vue") },
	{ name: "digitalClock", load: () => import("./digital-clock.vue") },
	{ name: "instanceCloud", load: () => import("./instance-cloud.vue") },
	{ name: "postForm", load: () => import("./post-form.vue") },
	{ name: "userList", load: () => import("./user-list.vue") },
	{ name: "serverInfo", load: () => import("./server-info.vue") },
];

const components = new Map<string, Component>(
	loaders.map((loader) => [loader.name, defineAsyncComponent(loader.load)]),
);

/** Firefish's `widgets` export: the names offered by the picker. */
export const widgets: string[] = loaders.map((loader) => loader.name);

export function widgetComponent(name: string): Component | null {
	return components.get(name) ?? null;
}

export function isWidgetName(name: string): boolean {
	return components.has(name);
}

/** Every picker entry resolves through the `_widgets.<name>` dictionary key. */
export function widgetTitle(name: string): string {
	return t(`_widgets.${name}`);
}
