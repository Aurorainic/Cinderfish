// Ported from firefish@b425f61:packages/client/src/store.ts (AGPL-3.0-only)
//
// Compatibility shim. The real definition lives in ./base.ts, which owns the
// single "base" Storage instance after the second-wave merge. This module only
// re-exports it so that first-wave imports of "@/store/store" and Firefish-style
// imports of "@/store" (./index.ts) all receive the same object.
export {
	defaultReactions,
	defaultStore,
	menuOptions,
	noteActions,
	notePostInterruptors,
	noteViewInterruptors,
	postFormActions,
	setStoreAccount,
	store,
	userActions,
} from "./base";
export type {
	BaseStoreDefinition,
	MenuOption,
	NoteVisibilityPreference,
	StatusBarItem,
	StatusbarItem,
	Store,
	TimelineMemory,
	TimelineSource,
	WidgetItem,
} from "./base";
