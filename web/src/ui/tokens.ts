// Shell wiring between App.vue and main.ts.
//
// App.vue must not import the store or the account layer directly (the shell
// choice belongs to the entry point), so main.ts provides this registry and
// App.vue injects it. Shells are async components, cached per mode.
import type { Component, InjectionKey, Ref } from "vue";

export type ShellMode = "universal" | "deck" | "zen" | "visitor";

export interface ShellRegistry {
  mode: Readonly<Ref<ShellMode>>;
  load: (mode: ShellMode) => Component;
}

export const ShellKey: InjectionKey<ShellRegistry> = Symbol("cinderfish:shell");

/** Firefish's `ui` preference (localStorage), extended with the visitor shell. */
export const SHELL_STORAGE_KEY = "ui";

// Route metadata consumed by the shell switch in main.ts. `requiresLogin` is
// declared by the account layer's guard module; declaring it here as well is
// harmless interface merging and keeps routes.ts valid if that module moves.
declare module "vue-router" {
  interface RouteMeta {
    requiresLogin?: boolean;
    /** "home" follows the stored preference; "deck" pins the deck shell. */
    shell?: "home" | "deck";
  }
}
