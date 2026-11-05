// Ported from firefish@b425f61:packages/client/src/components/index.ts (AGPL-3.0-only)
//
// Firefish lists every global component by hand and then declares them for
// Vetur. The list is fixed by the port contract (section 4), so a glob over
// components/global/*.vue registers exactly those files without another import
// table to keep in sync: MkA, MkAcct, MkAvatar, MkEmoji, MkUserName,
// MkEllipsis, MkTime, MkUrl, MkLoading, MkError, MkPageHeader, MkSpacer,
// MkStickyContainer. MkMisskeyFlavoredMarkdown.vue gets the extra "Mfm" alias.
import type { App, Component } from "vue";

type GlobModule = { default: Component };

const globals = import.meta.glob<GlobModule>("./global/*.vue", { eager: true });

function componentName(path: string): string | null {
	const file = path.split("/").pop();
	if (file == null) return null;
	return file.replace(/.vue$/, "");
}

export function install(app: App): void {
	for (const [path, module] of Object.entries(globals)) {
		const name = componentName(path);
		if (name == null || module.default == null) continue;
		app.component(name, module.default);
	}

	// Firefish registers the MFM renderer as <Mfm>.
	const mfm = globals["./global/MkMisskeyFlavoredMarkdown.vue"];
	if (mfm?.default != null) app.component("Mfm", mfm.default);
}

export default install;
