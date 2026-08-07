import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

export default defineConfig({
  plugins: [react()],
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
        url: "http://localhost:5173",
      },
    },
    setupFiles: "./vitest.setup.ts",
    css: true,
  },
});
