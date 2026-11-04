// Application entry point (index.html loads /src/main.ts into #app).
import "./styles/index.scss";

import { computed, createApp, defineAsyncComponent, markRaw, type Component } from "vue";
import App from "./App.vue";
import { me, refreshAccount, restoreSession } from "./account";
import { initTheme } from "./composables/use-theme";
import { router } from "./router";
import { store } from "./store/base";
import { ShellKey, SHELL_STORAGE_KEY, type ShellMode, type ShellRegistry } from "./ui/tokens";

// Apply the persisted theme before the first paint of the SPA (index.html has
// already set data-theme to avoid a flash of the wrong background).
initTheme();

function readUiPreference(): string | null {
  try {
    return localStorage.getItem(SHELL_STORAGE_KEY);
  } catch {
    return null;
  }
}

const shellMode = computed<ShellMode>(() => {
  const route = router.currentRoute.value;

  // The /deck route pins the deck shell regardless of the stored preference.
  if (route.meta.shell === "deck") return "deck";

  // Logged-out visitors get the chrome-light layout.
  if (me.value == null) return "visitor";

  if (route.query.zen != null) return "zen";

  const ui = readUiPreference();
  if (ui === "zen") return "zen";
  if (ui === "deck") return "deck";
  if (store.state.deck != null) return "deck";

  return "universal";
});

const shellLoaders: Record<ShellMode, () => Promise<{ default: Component }>> = {
  universal: () => import("./ui/UniversalShell.vue"),
  deck: () => import("./ui/DeckShell.vue"),
  zen: () => import("./ui/ZenShell.vue"),
  visitor: () => import("./ui/VisitorShell.vue"),
};

const shellCache = new Map<ShellMode, Component>();

const registry: ShellRegistry = {
  mode: shellMode,
  load(mode) {
    const cached = shellCache.get(mode);
    if (cached != null) return cached;
    const component = markRaw(defineAsyncComponent(shellLoaders[mode]));
    shellCache.set(mode, component);
    return component;
  },
};

const app = createApp(App);
app.provide(ShellKey, registry);
app.use(router);
app.mount("#app");

// Rehydrate the session from storage, then refresh the cached profile in the
// background; a stale token simply leaves the stored object in place until the
// next API call clears it.
void restoreSession().catch(() => undefined);
if (me.value != null) {
  void refreshAccount().catch(() => undefined);
}
