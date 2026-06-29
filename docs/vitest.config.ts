import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import path from "path";

export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: [path.resolve(__dirname, "vitest.setup.ts")],
    root: __dirname,
    include: ["__tests__/**/*.{test,spec}.{ts,tsx}"],
    exclude: ["node_modules/**", ".next/**", "out/**"],
  },
  resolve: {
    // The Pagefind runtime bundle only exists after the `pagefind` postbuild
    // step. Point the lazy `/pagefind/pagefind.js` import at a stub so the
    // Search component can be unit-tested without a built index.
    alias: {
      "/pagefind/pagefind.js": path.resolve(__dirname, "test/pagefind-stub.ts"),
    },
  },
});
