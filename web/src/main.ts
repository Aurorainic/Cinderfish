import { createApp } from "vue";
import App from "./App.vue";
import { router } from "./router";
import { applyTheme, resolveTheme } from "./stores/theme";
import { loadInstance } from "./stores/instance";
import { applyFontScale } from "./views/settings/font-scale";
// Initialises the Firefish-shaped `me` projection (web/src/me.ts) from the
// session; ported components import it directly, this import keeps it live
// from startup rather than from the first component that happens to load it.
import "./me";
import "./styles/main.css";
import "./styles/mfm.css";

if (!document.documentElement.dataset.theme) {
  applyTheme(resolveTheme(), { persist: false });
}

applyFontScale();

// The shell brand, the composer's length limit and the Firefish instance
// projection all read this store, so start the one metadata request here.
void loadInstance();

createApp(App).use(router).mount("#app");
