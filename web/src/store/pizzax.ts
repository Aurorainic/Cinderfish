import { getCurrentInstance, onUnmounted, ref, watch, type Ref } from "vue";
import { api } from "../api/client";

/**
 * PIZZAX — Firefish's lightweight preference store, ported to TypeScript.
 *
 * Three persistence scopes, same as Firefish:
 *   device        localStorage, shared by every account on this browser
 *   deviceAccount localStorage, namespaced by account id
 *   account       server registry (`i/registry/*`), cached in localStorage
 *
 * Keys are identical to Firefish's (`pizzax::base`, `pizzax::deck`, ...) so a
 * future port can read existing data. All writes are mirrored across tabs with
 * a BroadcastChannel.
 */

export type StoreWhere = "account" | "device" | "deviceAccount";

export interface StoreField<T = unknown> {
	where: StoreWhere;
	default: T;
}

export type StateDef = Record<string, StoreField>;

/** Server-side registry adapter for `account`-scoped keys. */
export interface PizzaxRemote {
	fetchAll: (bucket: string) => Promise<Record<string, unknown>>;
	set: (bucket: string, key: string, value: unknown) => Promise<void> | void;
}

export interface PizzaxOptions {
	/** Current account id; defaults to the `account` localStorage object. */
	getUserId?: () => string | null;
	/** Override the registry transport (tests, alternate backends). */
	remote?: PizzaxRemote | null;
}

type StoreDefaults<T extends StateDef> = {
	[K in keyof T]: T[K]["default"];
};

type ReactiveState<T extends StateDef> = {
	[K in keyof T]: Ref<T[K]["default"]>;
};

type ArrayElement<A> = A extends readonly (infer E)[] ? E : never;

type ArrayKeys<T extends StateDef> = {
	[K in keyof T]: T[K]["default"] extends readonly unknown[] ? K : never;
}[keyof T] & string;

const BROADCAST_PREFIX = "pizzax:bc:";

function accountId(): string | null {
	try {
		const raw = localStorage.getItem("account");
		if (raw == null) return null;
		const parsed = JSON.parse(raw) as { id?: unknown };
		return typeof parsed.id === "string" ? parsed.id : null;
	} catch {
		return null;
	}
}

function readJson(raw: string | null): Record<string, unknown> {
	if (raw == null) return {};
	try {
		const parsed = JSON.parse(raw) as unknown;
		return parsed != null && typeof parsed === "object"
			? (parsed as Record<string, unknown>)
			: {};
	} catch {
		return {};
	}
}

let tabId = Math.random().toString(36).slice(2);

export class Storage<T extends StateDef> {
	readonly key: string;
	readonly keyForLocalStorage: string;
	readonly def: T;
	readonly state: StoreDefaults<T>;
	readonly reactiveState: ReactiveState<T>;

	private channel: BroadcastChannel | null = null;
	private readonly options: PizzaxOptions;

	constructor(key: string, def: T, options: PizzaxOptions = {}) {
		this.key = key;
		this.keyForLocalStorage = `pizzax::${key}`;
		this.def = def;
		this.options = options;

		const uid = this.userId();
		const deviceState = readJson(localStorage.getItem(this.keyForLocalStorage));
		const deviceAccountState =
			uid == null
				? {}
				: readJson(localStorage.getItem(`${this.keyForLocalStorage}::${uid}`));
		const registryCache =
			uid == null
				? {}
				: readJson(
						localStorage.getItem(
							`${this.keyForLocalStorage}::cache::${uid}`,
						),
					);

		const state: Record<string, unknown> = {};
		const reactiveState: Record<string, Ref<unknown>> = {};

		for (const [k, v] of Object.entries(def)) {
			if (v.where === "device" && k in deviceState) {
				state[k] = deviceState[k];
			} else if (v.where === "deviceAccount" && k in deviceAccountState) {
				state[k] = deviceAccountState[k];
			} else if (v.where === "account" && k in registryCache) {
				state[k] = registryCache[k];
			} else {
				state[k] = v.default;
			}
			reactiveState[k] = ref(state[k]);
		}

		this.state = state as StoreDefaults<T>;
		this.reactiveState = reactiveState as ReactiveState<T>;

		this.setupBroadcast(uid);
		if (uid != null) {
			this.syncFromRegistry(uid);
		}
	}

	/**
	 * Account-scoped values live on the server; the localStorage copy is only a
	 * cache so the first paint does not flash defaults.
	 */
	private userId(): string | null {
		return this.options.getUserId?.() ?? accountId();
	}

	private fetchAllRemote(): Promise<Record<string, unknown>> {
		if (this.options.remote != null) {
			return this.options.remote.fetchAll(this.key);
		}
		return api<Record<string, unknown>>(
			"i/registry/get-all",
			{ scope: ["client", this.key] },
			undefined,
		);
	}

	private setRemote(key: string, value: unknown): void {
		const remote = this.options.remote;
		if (remote != null) {
			Promise.resolve(remote.set(this.key, key, value)).catch(() => {
				// Offline: the local cache remains authoritative for now.
			});
			return;
		}
		void api("i/registry/set", {
			scope: ["client", this.key],
			key,
			value,
		}).catch(() => {
			// Offline: the local cache remains authoritative for now.
		});
	}

	private syncFromRegistry(uid: string): void {
		window.setTimeout(() => {
			this.fetchAllRemote()
				.then((kvs) => {
					const cache: Record<string, unknown> = {};
					const state = this.state as Record<string, unknown>;
					const reactive = this.reactiveState as Record<string, Ref<unknown>>;
					for (const [k, v] of Object.entries(this.def)) {
						if (v.where !== "account") continue;
						if (k in kvs) {
							state[k] = kvs[k];
							reactive[k].value = kvs[k];
							cache[k] = kvs[k];
						}
					}
					localStorage.setItem(
						`${this.keyForLocalStorage}::cache::${uid}`,
						JSON.stringify(cache),
					);
				})
				.catch(() => {
					// Anonymous / offline: keep the cached or default values.
				});
		}, 1);
	}

	private setupBroadcast(uid: string | null): void {
		if (typeof BroadcastChannel === "undefined") return;
		this.channel = new BroadcastChannel(BROADCAST_PREFIX + this.key);
		this.channel.onmessage = (event: MessageEvent<unknown>) => {
			const message = event.data as {
				tab?: string;
				key?: string;
				value?: unknown;
				where?: StoreWhere;
				uid?: string | null;
			};
			if (message.tab === tabId || message.key == null) return;
			if (!(message.key in this.def)) return;
			if (message.where === "deviceAccount" && message.uid !== uid) return;
			if (message.where === "account" && message.uid !== uid) return;
			this.state[message.key as keyof T] = message.value as never;
			this.reactiveState[message.key as keyof T].value = message.value as never;
		};
	}

	private postBroadcast(key: string, value: unknown, where: StoreWhere): void {
		this.channel?.postMessage({
			tab: tabId,
			key,
			value,
			where,
			uid: this.userId(),
		});
	}

	set<K extends keyof T & string>(key: K, value: StoreDefaults<T>[K]): void {
		const field = this.def[key] as StoreField<StoreDefaults<T>[K]>;
		this.state[key] = value;
		this.reactiveState[key].value = value;

		const uid = this.userId();
		switch (field.where) {
			case "device": {
				const state = readJson(localStorage.getItem(this.keyForLocalStorage));
				state[key] = value;
				localStorage.setItem(
					this.keyForLocalStorage,
					JSON.stringify(state),
				);
				break;
			}
			case "deviceAccount": {
				if (uid == null) break;
				const state = readJson(
					localStorage.getItem(`${this.keyForLocalStorage}::${uid}`),
				);
				state[key] = value;
				localStorage.setItem(
					`${this.keyForLocalStorage}::${uid}`,
					JSON.stringify(state),
				);
				break;
			}
			case "account": {
				if (uid == null) break;
				const cache = readJson(
					localStorage.getItem(
						`${this.keyForLocalStorage}::cache::${uid}`,
					),
				);
				cache[key] = value;
				localStorage.setItem(
					`${this.keyForLocalStorage}::cache::${uid}`,
					JSON.stringify(cache),
				);
				this.setRemote(key, value);
				break;
			}
		}

		this.postBroadcast(key, value, field.where);
	}

	get<K extends keyof T & string>(key: K): StoreDefaults<T>[K] {
		return this.reactiveState[key].value;
	}

	push<K extends ArrayKeys<T>>(
		key: K,
		value: ArrayElement<StoreDefaults<T>[K]>,
	): void {
		const current = this.state[key] as unknown as unknown[];
		this.set(key, [...current, value] as unknown as StoreDefaults<T>[K]);
	}

	reset(key: keyof T & string): void {
		this.set(key, this.def[key].default as never);
	}

	/**
	 * Getter/setter pair for `v-model` in settings forms. The returned object
	 * is intentionally plain (not a ref), matching Firefish's usage.
	 */
	makeGetterSetter<K extends keyof T & string>(
		key: K,
		getter?: (value: StoreDefaults<T>[K]) => StoreDefaults<T>[K],
		setter?: (value: StoreDefaults<T>[K]) => StoreDefaults<T>[K],
	): { get: () => StoreDefaults<T>[K]; set: (value: StoreDefaults<T>[K]) => void } {
		const valueRef = ref(this.state[key]) as Ref<StoreDefaults<T>[K]>;

		const stop = watch(this.reactiveState[key], (value) => {
			valueRef.value = value;
		});

		if (getCurrentInstance() != null) {
			onUnmounted(() => stop());
		}

		return {
			get: () => (getter ? getter(valueRef.value) : valueRef.value),
			set: (value: StoreDefaults<T>[K]) => {
				const next = setter ? setter(value) : value;
				this.set(key, next);
				valueRef.value = next;
			},
		};
	}
}

/** Firefish calls this class `Storage`; the factory keeps call sites terse. */
export function createPizzax<T extends StateDef>(
	bucket: string,
	def: T,
	options: PizzaxOptions = {},
): Storage<T> {
	return new Storage(bucket, def, options);
}

export type { ReactiveState, StoreDefaults };
