import react from "@vitejs/plugin-react";
import UnoCSS from "unocss/vite";
import { defineConfig } from "vitest/config";

export default defineConfig({
  plugins: [
    react(),
    // Utilities and icons come from uno.config.ts, generated at build time.
    UnoCSS(),
  ],
  server: {
    port: 5173,
    proxy: {
      "/api": "http://localhost:8080",
    },
  },
  build: {
    rolldownOptions: {
      output: {
        codeSplitting: {
          groups: [
            {
              name: "lexical",
              test: /node_modules\/(?:@lexical|lexical)\//,
              includeDependenciesRecursively: false,
            },
            {
              name: "vendor",
              test: /node_modules/,
              includeDependenciesRecursively: false,
            },
          ],
        },
      },
    },
  },
  test: {
    environment: "jsdom",
    execArgv: ["--no-experimental-webstorage"],
    environmentOptions: {
      jsdom: {
        // A secure origin, because the session and CSRF cookies use the
        // __Host- prefix and are rejected by the browser without it.
        url: "https://localhost:5173",
      },
    },
    setupFiles: "./vitest.setup.ts",
    css: true,
    coverage: {
      provider: "v8",
      include: ["src/**/*.{ts,tsx}"],
      exclude: ["src/**/*.test.{ts,tsx}", "src/main.tsx"],
      reporter: ["text", "json-summary", "lcov"],
      thresholds: {
        statements: 75,
        branches: 72,
        functions: 68,
        lines: 78,
      },
    },
  },
});
