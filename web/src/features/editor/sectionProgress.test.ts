import { describe, expect, it } from "vitest";

import { storedDocumentFixture } from "../../test/harness";
import { countWords, documentProgress, sectionStatus } from "./sectionProgress";

function withText(text: string) {
  const stored = storedDocumentFixture();
  stored.content.sections[0]!.elements[0]!.richText = {
    root: {
      children: [{ type: "paragraph", children: [{ type: "text", text }] }],
    },
  } as never;
  return stored.content;
}

describe("sectionProgress", () => {
  it("conta palavras a partir do estado salvo, não do DOM", () => {
    const content = withText("A equipe identificou a necessidade");
    expect(countWords(content.sections[0]!)).toBe(5);
  });

  it("conta zero para uma seção ainda vazia", () => {
    const content = storedDocumentFixture().content;
    expect(countWords(content.sections[0]!)).toBe(0);
  });

  it("ignora espaço em excesso ao contar", () => {
    const content = withText("   dois    termos   ");
    expect(countWords(content.sections[0]!)).toBe(2);
  });

  it("descreve a posição e a obrigatoriedade da seção", () => {
    const content = withText("texto");
    const status = sectionStatus(content, content.sections[0]!);

    expect(status.number).toBe("1");
    expect(status.level).toBe(1);
    expect(status.required).toBe(true);
    expect(status.filled).toBe(true);
  });

  it("nomeia as seções obrigatórias ainda vazias", () => {
    const content = withText("texto");
    const progress = documentProgress(content);

    expect(progress.percent).toBe(50);
    expect(progress.missingRequired).toEqual(["Requisitos da contratação"]);
  });

  it("não cobra preenchimento de uma seção opcional", () => {
    const content = withText("texto");
    content.sections[1]!.required = false;

    const progress = documentProgress(content);

    expect(progress.missingRequired).toEqual([]);
  });
});
