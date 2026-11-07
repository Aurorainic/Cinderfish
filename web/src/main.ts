// Application entry point (index.html loads /src/main.ts into #app).
// Phosphor icon font, the two weights Firefish's UI uses (scripts/icon.ts
// emits "ph-bold" / "ph-fill" class names).
//
// The @font-face rules are imported from `styles/index.scss`, not here: a
// bare `import "@phosphor-icons/web/bold"` from the entry does emit the icon
// classes, but the package ships them in `src/bold/style.css`, whose
// `@font-face` ends up in a JS-injected stylesheet whose relative `url()`
// Vite cannot rewrite, so every glyph falls back to a system box.
import "./styles/index.scss";

import { computed, createApp, defineAsyncComponent, markRaw, type Component } from "vue";
import App from "./App.vue";
import { me, refreshAccount, restoreSession, updateMe } from "./account";
import { install as installComponents } from "./components";
import { initCustomCss } from "./composables/use-custom-css";
import { initTheme } from "./composables/use-theme";
import { install as installDirectives } from "./directives";
import { updateInstance } from "./instance";
import { router } from "./router";
import { store } from "./store/base";
import { stream } from "./stream";
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
// Global components (components/global/*.vue -> <MkA>, <Mfm>, ...) and the
// ported directives (v-appear / v-size / v-tooltip / ...). Both are installed
// before the router so the first rendered page already has them.
app.use(installDirectives);
app.use(installComponents);
app.provide(ShellKey, registry);
app.use(router);
app.mount("#app");

// Per-account custom CSS is applied right after mount; the store may replace
// the value again once the registry answers, and the watcher picks it up.
initCustomCss();

// Instance metadata (name / version / features) asynchronously; the reactive
// object in @/instance starts from the cached copy so first paint is correct.
void updateInstance();

// Rehydrate the session from storage, then refresh the cached profile in the
// background; a stale token simply leaves the stored object in place until the
// next API call clears it.
// Another tab's write (profile, pinned notes, follow or block counts) arrives on
// the main channel; Firefish's init.ts keeps `$i` in step the same way. The
// subscription waits for the session because the channel needs a credential.
void restoreSession()
  .catch(() => undefined)
  .then(() => {
    if (me.value == null) return;
    stream.useChannel("main").on("meUpdated", (fresh) => {
      if (me.value != null) updateMe(fresh);
    });
  });
if (me.value != null) {
  void refreshAccount().catch(() => undefined);
}
