import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// In dev the proxy makes /api same-origin with the Vite server, like mode A.
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/api": "http://localhost:8080",
    },
  },
});
