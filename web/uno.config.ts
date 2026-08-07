import presetIcons from "@unocss/preset-icons";
import presetWind4 from "@unocss/preset-wind4";
import { defineConfig } from "unocss";

/**
 * Utilities are generated from the project's own tokens, not from a stock
 * palette. A colour or spacing step that is not in `tokens.css` produces no
 * utility, which is what keeps the token system the source of truth instead of
 * turning every element into a place where new values can be invented.
 */
export default defineConfig({
  presets: [
    presetWind4({ preflights: { reset: false } }),
    presetIcons({
      scale: 1,
      // Icons are read from the locally installed collection at build time.
      // The Iconify runtime API is never contacted: it would fetch from an
      // external host, which the gateway's policy blocks and an institutional
      // installation cannot rely on.
      collections: {
        lucide: () =>
          import("@iconify-json/lucide/icons.json", {
            with: { type: "json" },
          }).then((module) => module.default),
      },
      extraProperties: { display: "inline-block", "vertical-align": "middle" },
    }),
  ],
  theme: {
    colors: {
      canvas: "var(--color-canvas)",
      panel: "var(--color-panel)",
      paper: "var(--color-paper)",
      ink: "var(--color-ink)",
      text: "var(--color-text)",
      subtle: "var(--color-text-subtle)",
      line: "var(--color-border)",
      "line-strong": "var(--color-border-strong)",
      accent: "var(--color-accent)",
      "accent-strong": "var(--color-accent-strong)",
      "accent-soft": "var(--color-accent-soft)",
      "accent-line": "var(--color-accent-border)",
      success: "var(--color-success)",
      "success-soft": "var(--color-success-soft)",
      danger: "var(--color-danger)",
    },
    spacing: {
      1: "var(--space-1)",
      2: "var(--space-2)",
      3: "var(--space-3)",
      4: "var(--space-4)",
      5: "var(--space-5)",
      6: "var(--space-6)",
    },
    radius: {
      1: "var(--radius-1)",
      2: "var(--radius-2)",
      3: "var(--radius-3)",
      4: "var(--radius-4)",
    },
    font: {
      interface: "var(--font-interface)",
      document: "var(--font-document)",
    },
  },
  shortcuts: {
    // Promoted only after the pattern appeared more than once.
    "field-label": "text-11px font-600 tracking-wide uppercase text-subtle",
    "row-between": "flex items-center justify-between gap-2",
  },
});
