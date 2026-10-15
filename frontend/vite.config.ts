import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// The build writes into the Go server's embedded assets directory, so the
// dashboard ships inside the pathwise binary. In dev, /api is proxied to a
// running `pathwise serve`.
export default defineConfig({
  plugins: [react()],
  base: "/",
  build: {
    outDir: "../internal/server/assets",
    emptyOutDir: true,
  },
  server: {
    proxy: {
      "/api": "http://127.0.0.1:7777",
    },
  },
});
