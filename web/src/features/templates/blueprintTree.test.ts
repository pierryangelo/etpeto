import { describe, expect, it } from "vitest";

import type { BlueprintSection } from "../../shared/api/client";
import {
  addSection,
  canIndent,
  canOutdent,
  collectKeys,
  depthOf,
  deriveKey,
  indentSection,
  moveSection,
  newSection,
  outdentSection,
  removeSection,
  updateSection,
} from "./blueprintTree";

function section(
  key: string,
  children: BlueprintSection[] = [],
): BlueprintSection {
  return {
    key,
    title: key,
    required: false,
    allowRename: false,
    allowedElements: ["richText"],
    ...(children.length > 0 ? { sections: children } : {}),
  };
}

function tree(): BlueprintSection[] {
  return [
    section("necessidade"),
    section("requisitos", [section("tecnicos"), section("sustentabilidade")]),
    section("conclusao"),
  ];
}

function shape(sections: BlueprintSection[]): string {
  return sections
    .map((item) => {
      const children = item.sections ?? [];
      return children.length ? `${item.key}(${shape(children)})` : item.key;
    })
    .join(",");
}

describe("blueprintTree", () => {
  it("não altera a árvore recebida", () => {
    const original = tree();
    const snapshot = shape(original);

    addSection(original, null, newSection("Extra"));
    removeSection(original, "conclusao");
    moveSection(original, "conclusao", "up");

    expect(shape(original)).toBe(snapshot);
  });

  it("deriva uma chave estável a partir do título", () => {
    expect(deriveKey("Pesquisa de preços", new Set())).toBe(
      "pesquisa-de-precos",
    );
    expect(deriveKey("  Conclusão!  ", new Set())).toBe("conclusao");
    expect(deriveKey("", new Set())).toBe("secao");
  });

  it("evita repetir uma chave já usada em qualquer nível", () => {
    const keys = collectKeys(tree());
    expect(deriveKey("tecnicos", keys)).toBe("tecnicos-2");
  });

  it("acrescenta seção na raiz e subseção dentro da mãe", () => {
    const withRoot = addSection(tree(), null, newSection("Anexos"));
    expect(shape(withRoot)).toBe(
      "necessidade,requisitos(tecnicos,sustentabilidade),conclusao,anexos",
    );

    const withChild = addSection(tree(), "requisitos", newSection("Garantia"));
    expect(shape(withChild)).toBe(
      "necessidade,requisitos(tecnicos,sustentabilidade,garantia),conclusao",
    );
  });

  it("remove uma subseção sem tocar nas irmãs", () => {
    expect(shape(removeSection(tree(), "tecnicos"))).toBe(
      "necessidade,requisitos(sustentabilidade),conclusao",
    );
  });

  it("remove a mãe e, com ela, a subárvore", () => {
    expect(shape(removeSection(tree(), "requisitos"))).toBe(
      "necessidade,conclusao",
    );
  });

  it("edita os campos mas nunca a chave", () => {
    const edited = updateSection(tree(), "tecnicos", {
      title: "Requisitos técnicos",
      required: true,
      key: "outra-chave",
    });

    const found = edited[1]!.sections![0]!;
    expect(found.title).toBe("Requisitos técnicos");
    expect(found.required).toBe(true);
    // Renaming must not change what the section means for existing documents.
    expect(found.key).toBe("tecnicos");
  });

  it("reordena entre irmãs, em qualquer nível", () => {
    expect(shape(moveSection(tree(), "conclusao", "up"))).toBe(
      "necessidade,conclusao,requisitos(tecnicos,sustentabilidade)",
    );
    expect(shape(moveSection(tree(), "sustentabilidade", "up"))).toBe(
      "necessidade,requisitos(sustentabilidade,tecnicos),conclusao",
    );
  });

  it("ignora um movimento que sairia da lista", () => {
    expect(shape(moveSection(tree(), "necessidade", "up"))).toBe(shape(tree()));
    expect(shape(moveSection(tree(), "conclusao", "down"))).toBe(shape(tree()));
  });

  it("transforma seção em subseção da irmã anterior", () => {
    expect(shape(indentSection(tree(), "conclusao"))).toBe(
      "necessidade,requisitos(tecnicos,sustentabilidade,conclusao)",
    );
    expect(shape(indentSection(tree(), "sustentabilidade"))).toBe(
      "necessidade,requisitos(tecnicos(sustentabilidade)),conclusao",
    );
  });

  it("recusa recuar a primeira da lista, que não tem irmã acima", () => {
    expect(shape(indentSection(tree(), "necessidade"))).toBe(shape(tree()));
    expect(canIndent(tree(), "necessidade")).toBe(false);
    expect(canIndent(tree(), "conclusao")).toBe(true);
  });

  it("recusa um recuo que ultrapassaria a profundidade aceita", () => {
    const deep = [
      section("a"),
      section("b", [section("c", [section("d", [section("e")])])]),
    ];
    // b já tem quatro níveis; recuar b passaria de quatro.
    expect(shape(indentSection(deep, "b"))).toBe(shape(deep));
    expect(canIndent(deep, "b")).toBe(false);
  });

  it("promove a subseção para logo depois da antiga mãe", () => {
    expect(shape(outdentSection(tree(), "tecnicos"))).toBe(
      "necessidade,requisitos(sustentabilidade),tecnicos,conclusao",
    );
  });

  it("leva junto a subárvore ao promover", () => {
    const nested = [section("a", [section("b", [section("c")])])];
    expect(shape(outdentSection(nested, "b"))).toBe("a,b(c)");
  });

  it("não promove o que já está na raiz", () => {
    expect(canOutdent(tree(), "necessidade")).toBe(false);
    expect(canOutdent(tree(), "tecnicos")).toBe(true);
  });

  it("informa a profundidade de cada seção", () => {
    expect(depthOf(tree(), "necessidade")).toBe(1);
    expect(depthOf(tree(), "tecnicos")).toBe(2);
    expect(depthOf(tree(), "inexistente")).toBe(0);
  });
});
