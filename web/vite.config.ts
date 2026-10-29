import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

// The dev server proxies everything the Go backend owns so the SPA can
// run on :5173 while the API stays on :3000.
export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      "/api": "http://127.0.0.1:3000",
      "/media": "http://127.0.0.1:3000",
      "/oauth": "http://127.0.0.1:3000",
      "/.well-known": "http://127.0.0.1:3000",
      "/nodeinfo": "http://127.0.0.1:3000",
      "/streaming": { target: "ws://127.0.0.1:3000", ws: true },
    },
  },
  build: {
    outDir: "dist",
    sourcemap: false,
  },
});
