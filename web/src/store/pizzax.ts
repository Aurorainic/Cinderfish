// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/pizzax.ts
//
// Plugin-free subset of Firefish's pizzax storage. Differences from upstream:
//  - the Streaming connection and `i/registry` cloud sync are gone; Cinderfish
//    has no registry endpoint, so `account` state falls back to a per-account
//    localStorage bucket (see TODO below).
//  - `me` is read from web/src/me.ts (a live binding projected from the
//    session store) instead of Firefish's account module.
import { ref, watch, onUnmounted, type Ref } from "vue";
import { isSignedIn, me } from "../me";

export type StateDef = Record<
  string,
  {
    where: "account" | "device" | "deviceAccount";
    default: unknown;
  }
>;

type ArrayElement<A> = A extends readonly (infer T)[] ? T : never;

// localStorage may be unavailable (private mode, SSR test env); degrade to an
// in-memory bucket so the store still works for the session.
export type SimpleStorage = Pick<StorageLike, "getItem" | "setItem">;
type StorageLike = typeof globalThis extends { localStorage: infer L } ? L : never;
// Fallback shape when localStorage is absent; matches the DOM Storage methods.
const memoryStorage = new Map<string, string>();
function backingStorage(): SimpleStorage {
  if (typeof localStorage !== "undefined") return localStorage;
  return {
    getItem: (key) => memoryStorage.get(key) ?? null,
    setItem: (key, value) => memoryStorage.set(key, value),
  };
}

function readJson(storage: SimpleStorage, key: string): Record<string, unknown> {
  try {
    const raw = storage.getItem(key);
    if (!raw) return {};
    const parsed: unknown = JSON.parse(raw);
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>;
    }
  } catch {
    // Corrupt entry behaves as if absent.
  }
  return {};
}

function writeJson(
  storage: SimpleStorage,
  key: string,
  value: Record<string, unknown>,
): void {
  try {
    storage.setItem(key, JSON.stringify(value));
  } catch {
    // Quota or serialization failure keeps the value for this session only.
  }
}

export class Storage<T extends StateDef> {
  public readonly key: string;
  public readonly keyForLocalStorage: string;
  public readonly def: T;

  public readonly state: { [K in keyof T]: T[K]["default"] };
  public readonly reactiveState: { [K in keyof T]: Ref<T[K]["default"]> };

  constructor(key: string, def: T) {
    this.key = key;
    this.keyForLocalStorage = `pizzax::${key}`;
    this.def = def;

    const storage = backingStorage();
    const deviceState = readJson(storage, this.keyForLocalStorage);
    // TODO(missing capability): Cinderfish has no i/registry backend, so
    // "account" state degrades to a per-account device bucket in localStorage
    // instead of a server-side registry with a local cache.
    const deviceAccountState = isSignedIn(me)
      ? readJson(storage, `${this.keyForLocalStorage}::${me.id}`)
      : {};

    const state = {} as Record<string, unknown>;
    const reactiveState = {} as Record<string, Ref<unknown>>;
    for (const [k, v] of Object.entries(def)) {
      if (v.where === "device" && Object.prototype.hasOwnProperty.call(deviceState, k)) {
        state[k] = deviceState[k];
      } else if (
        (v.where === "deviceAccount" || v.where === "account") &&
        isSignedIn(me) &&
        Object.prototype.hasOwnProperty.call(deviceAccountState, k)
      ) {
        state[k] = deviceAccountState[k];
      } else {
        state[k] = v.default;
      }
    }
    for (const [k, v] of Object.entries(state)) {
      reactiveState[k] = ref(v);
    }
    this.state = state as typeof this.state;
    this.reactiveState = reactiveState as typeof this.reactiveState;
  }

  public set<K extends keyof T>(key: K & string, value: T[K]["default"]): void {
    this.state[key] = value;
    this.reactiveState[key].value = value;

    const storage = backingStorage();
    switch (this.def[key].where) {
      case "device": {
        const deviceState = readJson(storage, this.keyForLocalStorage);
        deviceState[key] = value;
        writeJson(storage, this.keyForLocalStorage, deviceState);
        break;
      }
      case "deviceAccount":
      // TODO(missing capability): same registry fallback as above — "account"
      // is persisted per account in localStorage, not on the server.
      case "account": {
        if (!isSignedIn(me)) break;
        const bucketKey = `${this.keyForLocalStorage}::${me.id}`;
        const deviceAccountState = readJson(storage, bucketKey);
        deviceAccountState[key] = value;
        writeJson(storage, bucketKey, deviceAccountState);
        break;
      }
    }
  }

  public push<K extends keyof T & string>(
    key: K,
    value: ArrayElement<T[K]["default"]>,
  ): void {
    const currentState = this.state[key] as unknown[];
    this.set(key, [...currentState, value] as T[K]["default"]);
  }

  public reset(key: keyof T & string): void {
    this.set(key, this.def[key].default);
  }

  /**
   * A small getter/setter pair for a key, mainly for settings controls used
   * as a v-model inside Vue components.
   */
  public makeGetterSetter<K extends keyof T>(
    key: K & string,
    getter?: (oldV: T[K]["default"]) => T[K]["default"],
    setter?: (oldV: T[K]["default"]) => T[K]["default"],
  ) {
    const valueRef = ref(this.state[key]) as Ref<T[K]["default"]>;

    const stop = watch(this.reactiveState[key], (val) => {
      valueRef.value = val;
    });

    // Only stops when called inside a Vue component; same leak as upstream.
    onUnmounted(() => {
      stop();
    });

    return {
      get: () => (getter ? getter(valueRef.value) : valueRef.value),
      set: (value: T[K]["default"]) => {
        const val = setter ? setter(value) : value;
        this.set(key, val);
        valueRef.value = val;
      },
    };
  }
}
