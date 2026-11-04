import { createPizzax } from "./pizzax";
import type { Widget } from "./index";

/**
 * Widget deck preferences, ported from Firefish's `defaultStore.widgets`
 * usage in `ui/universal.widgets.vue`. The widget list itself lives in the
 * `base` bucket (`defaultStore.widgets`) for compatibility; this bucket keeps
 * widget-deck-only presentation state.
 */

export const widgetStore = createPizzax("widgets", {
	editMode: { where: "device", default: false },
	addColumnOnMobile: { where: "device", default: true },
});

export type { Widget };
