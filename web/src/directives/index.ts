// Ported from firefish@b425f61:packages/client/src/directives/index.ts (AGPL-3.0-only)
//
// Firefish ships 13 directives; the five below are the ones this wave's
// components use. Further directives can be added to this switch when the
// component that needs them is ported.
import type { App } from "vue";
import adaptiveBorder from "./adaptive-border";
import appear from "./appear";
import hotkey from "./hotkey";
import size from "./size";
import tooltip from "./tooltip";
import userPreview from "./user-preview";

export function install(app: App): void {
	app.directive("adaptive-border", adaptiveBorder);
	app.directive("appear", appear);
	app.directive("size", size);
	app.directive("tooltip", tooltip);
	app.directive("userPreview", userPreview);
	app.directive("user-preview", userPreview);
	app.directive("hotkey", hotkey);
}

export default install;
