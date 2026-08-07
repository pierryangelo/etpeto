import react from "@vitejs/plugin-react";
import Icons from "unplugin-icons/vite";
import { defineConfig } from "vitest/config";

export default defineConfig({
  plugins: [
    react(),
    // Icons are compiled into the bundle as inline SVG. The Iconify runtime API
    // is deliberately not used: it would fetch from an external host, which the
    // gateway's content security policy blocks and an institutional install
    // cannot depend on.
    Icons({ compiler: "jsx", jsx: "react", scale: 1 }),
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
