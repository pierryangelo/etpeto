import { readdirSync, readFileSync, statSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

/**
 * The rule that keeps the design system from eroding.
 *
 * A screen composes primitives and never decides appearance. Without a check,
 * the next screen quietly invents its own button — which is exactly how this
 * codebase ended up with four of them.
 *
 * Layout is not appearance: a feature still says how its own boxes sit next to
 * each other. What it may not do is choose colour, border, radius, shadow or
 * type size, because those belong to the system.
 */

const featureRoot = join(import.meta.dirname, "..", "..", "features");

// Each pattern names a decision that belongs to shared/ui or uno.config.ts.
const forbidden: { name: string; pattern: RegExp }[] = [
  {
    name: "cor de texto",
    pattern: /\btext-(ink|text|subtle|accent|danger|success)\b/,
  },
  {
    name: "cor de fundo",
    pattern: /\bbg-(paper|panel|canvas|accent|success|danger|line)\b/,
  },
  { name: "borda", pattern: /\bborder-(line|accent|danger|success)\b/ },
  { name: "tamanho de tipo", pattern: /\btext-\d+(\.\d+)?px\b/ },
  { name: "raio", pattern: /\brounded-\d/ },
  { name: "sombra", pattern: /\bshadow-\[/ },
];

// Files that legitimately carry system-level styling.
const allowed = new Set(["EditorWorkspace.module.css"]);

function sourceFiles(directory: string): string[] {
  return readdirSync(directory).flatMap((entry) => {
    const path = join(directory, entry);
    if (statSync(path).isDirectory()) return sourceFiles(path);
    if (!/\.tsx?$/.test(entry) || /\.test\.tsx?$/.test(entry)) return [];
    if (allowed.has(entry)) return [];
    return [path];
  });
}

describe("design system", () => {
  const files = sourceFiles(featureRoot);

  it("encontra os arquivos de tela para verificar", () => {
    expect(files.length).toBeGreaterThan(5);
  });

  it.each(forbidden)(
    "nenhuma tela decide $name por conta própria",
    ({ pattern }) => {
      const offenders = files
        .map((path) => ({ path, source: readFileSync(path, "utf8") }))
        .filter(({ source }) => pattern.test(source))
        .map(({ path }) => path.slice(featureRoot.length + 1));

      expect(
        offenders,
        `use um primitivo de shared/ui em vez de decidir aparência na tela`,
      ).toEqual([]);
    },
  );
});
