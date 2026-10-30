import { createApp } from "vue";
import App from "./App.vue";
import { router } from "./router";
import { applyTheme, resolveTheme } from "./stores/theme";
import "./styles/main.css";
import "./styles/mfm.css";

if (!document.documentElement.dataset.theme) {
  applyTheme(resolveTheme(), { persist: false });
}

createApp(App).use(router).mount("#app");
