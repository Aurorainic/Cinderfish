// Ported from firefish@b425f61:packages/client/src/ui/universal.widgets.vue (AGPL-3.0-only)
//
// The widget list itself is persisted under `defaultStore.state.widgets`
// (deviceAccount scope), exactly like Firefish's `defaultStore.widgets`. The
// mutations Firefish spreads across universal.widgets.vue and MkWidgets.vue are
// collected here so the widgets column and the deck column share one writer.
import { createPizzax } from "./pizzax";
import { defaultStore, type WidgetItem } from "./base";

/**
 * Widget-deck presentation state. The list of widgets lives in the `base`
 * bucket (`defaultStore.widgets`) for Firefish compatibility; this bucket only
 * keeps widget-column-only preferences.
 */
export const widgetStore = createPizzax("widgets", {
	editMode: { where: "device", default: false },
	addColumnOnMobile: { where: "device", default: true },
});

/** Firefish calls the list entry `Widget`; the base store calls it WidgetItem. */
export type Widget = WidgetItem;
export type WidgetData = Record<string, unknown>;

/** The reactive widget list. Mutations go through the helpers below. */
export function widgetList(): WidgetItem[] {
	return defaultStore.reactiveState.widgets.value;
}

function write(next: WidgetItem[]): void {
	defaultStore.set("widgets", next);
}

/** `uuid` is not a dependency; the platform generator has the same guarantee. */
export function createWidgetId(): string {
	const cryptoObject = globalThis.crypto as Crypto | undefined;
	if (cryptoObject != null && typeof cryptoObject.randomUUID === "function") {
		return cryptoObject.randomUUID();
	}
	return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
}

/** New widgets are prepended, as in Firefish (they land at the top). */
export function addWidget(name: string, data: WidgetData = {}): WidgetItem {
	const widget: WidgetItem = {
		name,
		id: createWidgetId(),
		place: null,
		data,
	};
	write([widget, ...widgetList()]);
	return widget;
}

export function removeWidget(id: string): void {
	write(widgetList().filter((widget) => widget.id !== id));
}

/**
 * Move a widget by one position. Firefish drags with vue-draggable-plus, which
 * is not a dependency here (contract §2), so ordering is done with up/down
 * buttons and a plain array swap.
 */
export function moveWidget(id: string, delta: number): void {
	const list = [...widgetList()];
	const index = list.findIndex((widget) => widget.id === id);
	if (index < 0) return;
	const target = index + delta;
	if (target < 0 || target >= list.length) return;
	const [moved] = list.splice(index, 1);
	list.splice(target, 0, moved);
	write(list);
}

export function updateWidgetData(id: string, data: WidgetData): void {
	write(
		widgetList().map((widget) =>
			widget.id === id ? { ...widget, data: { ...widget.data, ...data } } : widget,
		),
	);
}

export function setWidgets(widgets: WidgetItem[]): void {
	write(widgets);
}

export type { WidgetItem };
