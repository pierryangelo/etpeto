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
  /**
   * The design system lives here and in shared/ui.
   *
   * A shortcut holds the decision — what a secondary button *is* — and the
   * component in shared/ui holds the behaviour and the accessibility. A screen
   * composes primitives and never decides appearance, which is what stopped
   * every screen from inventing its own button.
   */
  shortcuts: {
    // Controls -------------------------------------------------------------
    "ds-control":
      "inline-flex items-center justify-center gap-1.5 rounded-1 border font-550 " +
      "transition-colors duration-140 disabled:opacity-45 disabled:cursor-default " +
      "focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-1 " +
      "focus-visible:outline-[var(--color-focus)]",
    "ds-btn": "ds-control h-7 px-2.5 text-12px cursor-pointer",
    "ds-btn-primary":
      "ds-btn border-accent-strong bg-accent text-white hover:not-disabled:bg-accent-strong",
    "ds-btn-secondary":
      "ds-btn border-line-strong bg-paper text-text " +
      "hover:not-disabled:border-accent-line hover:not-disabled:text-accent-strong",
    "ds-btn-quiet":
      "ds-btn border-transparent bg-transparent text-text hover:not-disabled:bg-panel",
    "ds-btn-danger":
      "ds-btn border-line bg-paper text-danger hover:not-disabled:border-danger",
    "ds-btn-sm": "h-6 px-2 text-11.5px",
    "ds-icon-btn":
      "ds-control size-6 rounded-1 border-transparent bg-transparent text-subtle " +
      "cursor-pointer hover:not-disabled:bg-panel hover:not-disabled:text-ink",

    // Inputs ---------------------------------------------------------------
    // font-inherit matters: a textarea does not inherit the family, which is
    // how the guidance field ended up rendering in a monospace face.
    "ds-input":
      "w-full rounded-1 border border-line bg-paper px-2 py-1.5 text-13px text-ink " +
      "font-[inherit] placeholder:text-subtle " +
      "focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-0 " +
      "focus-visible:outline-[var(--color-focus)] focus-visible:border-accent-line",
    "ds-textarea": "ds-input min-h-16 resize-y leading-relaxed",

    // Text -----------------------------------------------------------------
    "ds-label": "text-10.5px font-700 tracking-wide uppercase text-subtle",
    "ds-eyebrow": "text-10px font-700 tracking-wider uppercase text-subtle",
    "ds-help": "text-11px leading-snug text-subtle",
    "ds-title": "text-19px font-680 tracking-tight text-ink",
    "ds-meta": "text-11.5px text-subtle",

    // Surfaces -------------------------------------------------------------
    "ds-panel": "bg-panel border-line",
    "ds-paper": "bg-paper",
    "ds-divider": "w-px flex-none bg-line",
    "ds-row": "flex items-center gap-2",
    "ds-stack": "flex flex-col gap-3.5",

    // Status ---------------------------------------------------------------
    "ds-badge":
      "inline-block rounded-0.5 px-1.5 py-0.5 text-9.5px font-700 tracking-wide uppercase",
    "ds-badge-live": "ds-badge bg-success-soft text-success",
    "ds-badge-draft": "ds-badge bg-accent-soft text-accent-strong",
    "ds-badge-muted": "ds-badge bg-panel text-subtle",
  },
});
