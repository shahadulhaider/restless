import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import wails from "@wailsio/runtime/plugins/vite";

// https://vitejs.dev/config/
export default defineConfig({
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  build: {
    outDir: "../cmd/restless-app/dist",
    emptyOutDir: true,
    rollupOptions: {
      output: {
        manualChunks(id: string) {
          if (id.includes("monaco-editor/esm/vs/editor")) return "monaco-core";
          if (id.includes("monaco-editor/esm/vs/language")) return "monaco-lang";
          if (id.includes("monaco-editor/esm/vs/basic-languages")) return "monaco-basic-lang";
          if (id.includes("monaco-editor")) return "monaco-misc";
          if (id.includes("primevue") || id.includes("primeuix")) return "primevue";
        },
      },
    },
  },
  plugins: [vue(), wails("./bindings")],
});
