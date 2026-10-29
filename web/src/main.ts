import { createApp } from "vue";
import App from "./App.vue";
import { router } from "./router";
import "./styles/main.css";

// Resolve the theme before the first paint so the shell does not flash.
const stored = localStorage.getItem("cinderfish.theme");
const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
document.documentElement.dataset.theme =
  stored === "light" || stored === "dark" ? stored : prefersDark ? "dark" : "light";

createApp(App).use(router).mount("#app");
