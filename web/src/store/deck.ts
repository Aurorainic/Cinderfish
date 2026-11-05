import { markRaw } from "vue";
import { api } from "../api/client";
import type { NotificationType } from "../api/types";
import { createPizzax } from "./pizzax";

/**
 * Port of Firefish `packages/client/src/ui/deck/deck-store.ts`.
 *
 * The bucket (`deck`) and every key are unchanged. Column layout is stored in
 * the account registry under `["client", "deck", "profiles"]`, per profile.
 */

export interface ColumnWidget {
	name: string;
	id: string;
	data: Record<string, unknown>;
}

export interface Column {
	id: string;
	type:
		| "main"
		| "widgets"
		| "notifications"
		| "tl"
		| "antenna"
		| "channel"
		| "list"
		| "mentions"
		| "direct"
		// Cinderfish addition: `i/favorites` exists, so favourites get a column.
		| "favorites";
	name: string | null;
	width: number;
	widgets?: ColumnWidget[];
	active?: boolean;
	flexible?: boolean;
	antennaId?: string;
	channelId?: string;
	listId?: string;
	includingTypes?: NotificationType[];
	tl?: "home" | "local" | "social" | "recommended" | "global";
}

export const deckStore = markRaw(
	createPizzax("deck", {
		profile: {
			where: "deviceAccount" as const,
			default: "default",
		},
		columns: {
			where: "deviceAccount" as const,
			default: [] as Column[],
		},
		layout: {
			where: "deviceAccount" as const,
			default: [] as string[][],
		},
		columnAlign: {
			where: "deviceAccount" as const,
			default: "left" as "left" | "right" | "center",
		},
		alwaysShowMainColumn: {
			where: "deviceAccount" as const,
			default: true,
		},
		navWindow: {
			where: "deviceAccount" as const,
			default: true,
		},
	}),
);

const DECK_SCOPE = ["client", "deck", "profiles"];

function deepClone<T>(value: T): T {
	if (typeof structuredClone === "function") return structuredClone(value);
	return JSON.parse(JSON.stringify(value)) as T;
}

/** Column ids: Firefish imports `uuid`, which is not a dependency (contract §2). */
export function createDeckId(): string {
	const cryptoObject = globalThis.crypto as Crypto | undefined;
	if (cryptoObject != null && typeof cryptoObject.randomUUID === "function") {
		return cryptoObject.randomUUID();
	}
	return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
}

export async function loadDeck(): Promise<void> {
	let deck: { columns?: Column[]; layout?: string[][] };
	try {
		deck = await api<{ columns?: Column[]; layout?: string[][] }>(
			"i/registry/get",
			{ scope: DECK_SCOPE, key: deckStore.state.profile },
		);
	} catch (error) {
		const code =
			error != null && typeof error === "object" && "code" in error
				? (error as { code?: string }).code
				: null;
		if (code === "NO_SUCH_KEY") {
			// Switching profiles reloads in place, so the deck is cleared before
			// the empty entry is seeded — otherwise one profile's columns would
			// carry over into the next.
			deckStore.set("columns", []);
			deckStore.set("layout", []);
			if (deckStore.state.profile === "default") saveDeck();
			return;
		}
		throw error;
	}

	deckStore.set("columns", deck.columns ?? []);
	deckStore.set("layout", deck.layout ?? []);
}

let saveTimer: number | null = null;

/** Debounced registry write; 1s matches Firefish's throttle. */
export function saveDeck(): void {
	if (saveTimer != null) window.clearTimeout(saveTimer);
	saveTimer = window.setTimeout(() => {
		saveTimer = null;
		void api("i/registry/set", {
			scope: DECK_SCOPE,
			key: deckStore.state.profile,
			value: {
				columns: deckStore.state.columns,
				layout: deckStore.state.layout,
			},
		}).catch(() => {
			// Offline: the localStorage copy stays authoritative until the next write.
		});
	}, 1000);
}

export async function getProfiles(): Promise<string[]> {
	return api<string[]>("i/registry/keys", { scope: DECK_SCOPE });
}

export async function deleteProfile(key: string): Promise<void> {
	await api("i/registry/remove", { scope: DECK_SCOPE, key });
}

export async function renameProfile(oldKey: string, newKey: string): Promise<void> {
	if (oldKey === newKey) return;
	await api("i/registry/set", {
		scope: DECK_SCOPE,
		key: newKey,
		value: { columns: deckStore.state.columns, layout: deckStore.state.layout },
	});
	deckStore.set("profile", newKey);
	saveDeck();
	void deleteProfile(oldKey);
}

export function addColumn(column: Column): void {
	if (column.name === undefined) column.name = null;
	deckStore.push("columns", column);
	deckStore.push("layout", [column.id]);
	saveDeck();
}

export function removeColumn(id: string): void {
	deckStore.set(
		"columns",
		deckStore.state.columns.filter((column) => column.id !== id),
	);
	deckStore.set(
		"layout",
		deckStore.state.layout
			.map((ids) => ids.filter((item) => item !== id))
			.filter((ids) => ids.length > 0),
	);
	saveDeck();
}

export function swapColumn(a: string, b: string): void {
	const layout = deepClone(deckStore.state.layout);
	const aX = layout.findIndex((ids) => ids.includes(a));
	const aY = layout[aX]?.indexOf(a) ?? -1;
	const bX = layout.findIndex((ids) => ids.includes(b));
	const bY = layout[bX]?.indexOf(b) ?? -1;
	if (aX < 0 || bX < 0 || aY < 0 || bY < 0) return;
	layout[aX][aY] = b;
	layout[bX][bY] = a;
	deckStore.set("layout", layout);
	saveDeck();
}

function swapHorizontal(id: string, offset: -1 | 1): void {
	const layout = deepClone(deckStore.state.layout);
	const index = layout.findIndex((ids) => ids.includes(id));
	const target = layout[index + offset];
	if (index < 0 || target == null) return;
	layout[index + offset] = layout[index];
	layout[index] = target;
	deckStore.set("layout", layout);
	saveDeck();
}

export function swapLeftColumn(id: string): void {
	swapHorizontal(id, -1);
}

export function swapRightColumn(id: string): void {
	swapHorizontal(id, 1);
}

function swapVertical(id: string, offset: -1 | 1): void {
	const layout = deepClone(deckStore.state.layout);
	const index = layout.findIndex((ids) => ids.includes(id));
	if (index < 0) return;
	const ids = deepClone(layout[index]);
	const position = ids.indexOf(id);
	const target = ids[position + offset];
	if (position < 0 || target == null) return;
	ids[position + offset] = id;
	ids[position] = target;
	layout[index] = ids;
	deckStore.set("layout", layout);
	saveDeck();
}

export function swapUpColumn(id: string): void {
	swapVertical(id, -1);
}

export function swapDownColumn(id: string): void {
	swapVertical(id, 1);
}

export function stackLeftColumn(id: string): void {
	let layout = deepClone(deckStore.state.layout);
	const index = layout.findIndex((ids) => ids.includes(id));
	layout = layout.map((ids) => ids.filter((item) => item !== id));
	const target = layout[index - 1];
	if (index < 0 || target == null) return;
	target.push(id);
	layout = layout.filter((ids) => ids.length > 0);
	deckStore.set("layout", layout);
	saveDeck();
}

export function popRightColumn(id: string): void {
	let layout = deepClone(deckStore.state.layout);
	const index = layout.findIndex((ids) => ids.includes(id));
	const affected = layout[index];
	if (index < 0 || affected == null) return;
	layout = layout.map((ids) => ids.filter((item) => item !== id));
	layout.splice(index + 1, 0, [id]);
	layout = layout.filter((ids) => ids.length > 0);
	deckStore.set("layout", layout);

	const columns = deepClone(deckStore.state.columns);
	for (const column of columns) {
		if (affected.includes(column.id)) column.active = true;
	}
	deckStore.set("columns", columns);
	saveDeck();
}

export function addColumnWidget(id: string, widget: ColumnWidget): void {
	const columns = deepClone(deckStore.state.columns);
	const index = columns.findIndex((column) => column.id === id);
	const column = columns[index];
	if (column == null) return;
	if (column.widgets == null) column.widgets = [];
	column.widgets.unshift(widget);
	deckStore.set("columns", columns);
	saveDeck();
}

export function removeColumnWidget(id: string, widget: ColumnWidget): void {
	const columns = deepClone(deckStore.state.columns);
	const index = columns.findIndex((column) => column.id === id);
	const column = columns[index];
	if (column == null) return;
	column.widgets = (column.widgets ?? []).filter((item) => item.id !== widget.id);
	deckStore.set("columns", columns);
	saveDeck();
}

export function updateColumn(id: string, patch: Partial<Column>): void {
	const columns = deepClone(deckStore.state.columns);
	const index = columns.findIndex((column) => column.id === id);
	if (index < 0) return;
	columns[index] = { ...columns[index], ...patch };
	deckStore.set("columns", columns);
	saveDeck();
}
