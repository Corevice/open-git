import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import path from "path";

export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./vitest.setup.ts"],
    // docs/ is a separate package (excluded from the root tsconfig too) with its
    // own vitest config; it is tested via `cd docs && npm test`.
    exclude: ["node_modules/**", "e2e/**", ".next/**", "docs/**"],
    env: {
      NEXT_PUBLIC_API_BASE_URL: "http://localhost:8080",
      NEXT_PUBLIC_APP_VERSION: "test",
    },
    fakeTimers: {
      shouldAdvanceTime: true,
    },
  },
  resolve: {
    dedupe: ["react", "react-dom"],
    alias: {
      "@": path.resolve(__dirname, "."),
      // The published ESM build of libsodium-wrappers references a sibling
      // libsodium.mjs that is not shipped; use the self-contained CJS build.
      "libsodium-wrappers": path.resolve(
        __dirname,
        "node_modules/libsodium-wrappers/dist/modules/libsodium-wrappers.js",
      ),
    },
  },
});
