// Aggregation barrel for the store layer (Cinderfish-only; Firefish imports
// each module directly). The Firefish-derived modules re-export their own
// provenance headers.
export {
	ColdDeviceStorage,
	get as getCold,
	makeGetterSetter as makeColdGetterSetter,
	set as setCold,
	watch as watchCold,
	type ColdStoreShape,
	type PluginRecord,
	type SoundConfig,
} from "./cold-store";
export {
	addColumn,
	addColumnWidget,
	deckStore,
	deleteProfile,
	getProfiles,
	loadDeck,
	popRightColumn,
	removeColumn,
	removeColumnWidget,
	renameProfile,
	saveDeck,
	stackLeftColumn,
	swapColumn,
	swapDownColumn,
	swapLeftColumn,
	swapRightColumn,
	swapUpColumn,
	updateColumn,
	type Column,
	type ColumnWidget,
} from "./deck";
export {
	createLayoutState,
	LAYOUT_INJECTION_KEY,
	type LayoutMode,
	type LayoutPreference,
	type LayoutState,
} from "./layout";
export { createPizzax, Storage } from "./pizzax";
export type {
	PizzaxOptions,
	PizzaxRemote,
	ReactiveState,
	StateDef,
	StoreDefaults,
	StoreField,
	StoreWhere,
} from "./pizzax";
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
	type MenuOption,
	type NoteVisibilityPreference,
	type StatusbarItem,
	type TimelineMemory,
	type TimelineSource,
	type WidgetItem,
} from "./base";
/** Firefish calls the widget list entry Widget; alias kept for port parity. */
export type Widget = import("./base").WidgetItem;
