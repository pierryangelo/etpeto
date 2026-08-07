import { beforeEach, describe, expect, it } from "vitest";

import {
  clampOutline,
  clampPreview,
  defaultLayout,
  layoutStorageKey,
  legacyLayoutStorageKey,
  outlineBounds,
  previewBounds,
  readLayout,
  writeLayout,
} from "./panelLayout";

describe("panelLayout", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("mantém cada painel dentro dos limites utilizáveis", () => {
    expect(clampOutline(10)).toBe(outlineBounds.min);
    expect(clampOutline(9999)).toBe(outlineBounds.max);
    expect(clampOutline(260)).toBe(260);
    expect(clampPreview(10)).toBe(previewBounds.min);
    expect(clampPreview(9999)).toBe(previewBounds.max);
  });

  it("arredonda para pixel inteiro para não produzir meia coluna", () => {
    expect(clampOutline(260.4)).toBe(260);
    expect(clampPreview(301.6)).toBe(302);
  });

  it("guarda e restaura larguras e recolhimento", () => {
    writeLayout(localStorage, {
      outlineWidth: 300,
      previewWidth: 420,
      outlineOpen: false,
      previewOpen: true,
    });

    expect(readLayout(localStorage)).toEqual({
      outlineWidth: 300,
      previewWidth: 420,
      outlineOpen: false,
      previewOpen: true,
    });
  });

  it("corrige uma largura fora dos limites que tenha sido persistida", () => {
    localStorage.setItem(
      layoutStorageKey,
      JSON.stringify({
        outlineWidth: 1200,
        previewWidth: 10,
        outlineOpen: true,
        previewOpen: true,
      }),
    );

    const layout = readLayout(localStorage);

    expect(layout.outlineWidth).toBe(outlineBounds.max);
    expect(layout.previewWidth).toBe(previewBounds.min);
  });

  it("ignora conteúdo corrompido sem impedir a edição", () => {
    localStorage.setItem(layoutStorageKey, "{não é json");
    expect(readLayout(localStorage)).toEqual(defaultLayout());

    localStorage.setItem(
      layoutStorageKey,
      JSON.stringify({ outlineWidth: "larga" }),
    );
    expect(readLayout(localStorage)).toEqual(defaultLayout());
  });

  it("aproveita o recolhimento salvo pela versão anterior do layout", () => {
    localStorage.setItem(
      legacyLayoutStorageKey,
      JSON.stringify({ outlineOpen: false, previewOpen: false }),
    );

    const layout = readLayout(localStorage);

    expect(layout.outlineOpen).toBe(false);
    expect(layout.previewOpen).toBe(false);
    expect(layout.outlineWidth).toBe(outlineBounds.default);
  });

  it("não quebra quando o armazenamento recusa a escrita", () => {
    const refusing = {
      getItem: () => null,
      setItem: () => {
        throw new Error("quota exceeded");
      },
    } as unknown as Storage;

    expect(() => writeLayout(refusing, defaultLayout())).not.toThrow();
  });
});
