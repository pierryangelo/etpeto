import { describe, expect, it } from "vitest";

import workspaceCss from "./EditorWorkspace.module.css?raw";

/**
 * The workspace sets --outline-width and --preview-width inline, and the panels
 * consume them in CSS. jsdom does not evaluate the stylesheet, so a variable
 * that nobody reads looks fine in every component test while the divider moves
 * and the panel stays put — which is exactly what happened to the preview.
 */
describe("panel widths", () => {
  const widthVariables = ["--outline-width", "--preview-width"];

  it.each(widthVariables)("aplica %s a algum painel", (variable) => {
    const declarations = workspaceCss.match(
      new RegExp(`flex:[^;]*var\\(${variable}\\)`, "g"),
    );
    expect(
      declarations,
      `${variable} é definido mas nenhum painel o usa`,
    ).not.toBeNull();
  });

  it("mantém os painéis sem crescer ou encolher sozinhos", () => {
    // `flex: 0 0 <width>` is what pins a panel to its stored width; `1 1 auto`
    // would let the editor's content push it around.
    const pinned = workspaceCss.match(
      /flex:\s*0\s+0\s+var\(--\w[\w-]*-width\)/g,
    );
    expect(pinned).toHaveLength(widthVariables.length);
  });
});
