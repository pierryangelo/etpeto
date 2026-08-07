import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { DocumentSection } from "../../shared/api/client";
import { OutlineTree } from "./OutlineTree";

function nestedSections(): DocumentSection[] {
  return [
    {
      id: "necessidade",
      definitionKey: "necessidade",
      title: "Necessidade da contratação",
      required: true,
      elements: [],
    },
    {
      id: "requisitos",
      definitionKey: "requisitos",
      title: "Requisitos da contratação",
      required: true,
      elements: [],
      sections: [
        {
          id: "tecnicos",
          definitionKey: "requisitos-tecnicos",
          title: "Requisitos técnicos",
          required: true,
          elements: [],
          sections: [
            {
              id: "desempenho",
              definitionKey: "requisitos-desempenho",
              title: "Desempenho mínimo",
              required: false,
              elements: [],
            },
          ],
        },
        {
          id: "sustentabilidade",
          definitionKey: "requisitos-sustentabilidade",
          title: "Requisitos de sustentabilidade",
          required: false,
          elements: [],
        },
      ],
    },
  ];
}

function renderTree(activeSectionId = "necessidade") {
  const onSelect = vi.fn();
  render(
    <OutlineTree
      sections={nestedSections()}
      activeSectionId={activeSectionId}
      onSelect={onSelect}
    />,
  );
  return { onSelect };
}

describe("OutlineTree", () => {
  it("expõe o roteiro como árvore para tecnologia assistiva", () => {
    renderTree();

    expect(
      screen.getByRole("tree", { name: "Roteiro do documento" }),
    ).toBeVisible();
    const items = screen.getAllByRole("treeitem");
    expect(items).toHaveLength(5);
  });

  it("declara nível, posição e tamanho de cada ramo", () => {
    renderTree();

    const parent = screen.getByRole("treeitem", {
      name: /Requisitos da contratação/,
    });
    expect(parent).toHaveAttribute("aria-level", "1");
    expect(parent).toHaveAttribute("aria-posinset", "2");
    expect(parent).toHaveAttribute("aria-setsize", "2");

    const child = screen.getByRole("treeitem", { name: /Requisitos técnicos/ });
    expect(child).toHaveAttribute("aria-level", "2");
    expect(child).toHaveAttribute("aria-posinset", "1");
    expect(child).toHaveAttribute("aria-setsize", "2");

    const grandchild = screen.getByRole("treeitem", {
      name: /Desempenho mínimo/,
    });
    expect(grandchild).toHaveAttribute("aria-level", "3");
  });

  it("numera as seções pela posição na árvore", () => {
    renderTree();

    expect(screen.getByText("2.1")).toBeVisible();
    expect(screen.getByText("2.1.1")).toBeVisible();
    expect(screen.getByText("2.2")).toBeVisible();
  });

  it("marca ramos como expansíveis e permite recolher", async () => {
    const user = userEvent.setup();
    renderTree();

    const parent = screen.getByRole("treeitem", {
      name: /Requisitos da contratação/,
    });
    expect(parent).toHaveAttribute("aria-expanded", "true");

    await user.click(
      screen.getByRole("button", {
        name: "Recolher Requisitos da contratação",
      }),
    );

    expect(parent).toHaveAttribute("aria-expanded", "false");
    expect(
      screen.queryByRole("treeitem", { name: /Requisitos técnicos/ }),
    ).not.toBeInTheDocument();
  });

  it("uma folha não se apresenta como expansível", () => {
    renderTree();

    expect(
      screen.getByRole("treeitem", { name: /Necessidade da contratação/ }),
    ).not.toHaveAttribute("aria-expanded");
  });

  it("navega entre seções com as setas verticais", async () => {
    const user = userEvent.setup();
    renderTree();

    await user.tab();
    expect(screen.getByRole("treeitem", { name: /Necessidade/ })).toHaveFocus();

    await user.keyboard("{ArrowDown}");
    expect(
      screen.getByRole("treeitem", { name: /Requisitos da contratação/ }),
    ).toHaveFocus();

    await user.keyboard("{ArrowDown}");
    expect(
      screen.getByRole("treeitem", { name: /Requisitos técnicos/ }),
    ).toHaveFocus();

    await user.keyboard("{ArrowUp}");
    expect(
      screen.getByRole("treeitem", { name: /Requisitos da contratação/ }),
    ).toHaveFocus();
  });

  it("usa as setas horizontais para abrir e fechar o ramo", async () => {
    const user = userEvent.setup();
    renderTree("requisitos");

    const parent = screen.getByRole("treeitem", {
      name: /Requisitos da contratação/,
    });
    parent.focus();

    await user.keyboard("{ArrowLeft}");
    expect(parent).toHaveAttribute("aria-expanded", "false");

    await user.keyboard("{ArrowRight}");
    expect(parent).toHaveAttribute("aria-expanded", "true");

    // With the branch already open, ArrowRight moves into it.
    await user.keyboard("{ArrowRight}");
    expect(
      screen.getByRole("treeitem", { name: /Requisitos técnicos/ }),
    ).toHaveFocus();
  });

  it("mantém um único ponto de tabulação na árvore", async () => {
    const user = userEvent.setup();
    renderTree("tecnicos");

    await user.tab();

    // Tab reaches the active section, not the first one, so returning to the
    // outline puts focus where the person was working.
    expect(
      screen.getByRole("treeitem", { name: /Requisitos técnicos/ }),
    ).toHaveFocus();
  });

  it("seleciona a seção com Enter e com clique", async () => {
    const user = userEvent.setup();
    const { onSelect } = renderTree();

    await user.click(
      screen.getByRole("treeitem", { name: /Requisitos técnicos/ }),
    );
    expect(onSelect).toHaveBeenLastCalledWith("tecnicos");

    await user.keyboard("{Enter}");
    expect(onSelect).toHaveBeenCalledTimes(2);
  });

  it("assinala a seção em edição para leitores de tela", () => {
    renderTree("sustentabilidade");

    expect(
      screen.getByRole("treeitem", { name: /Requisitos de sustentabilidade/ }),
    ).toHaveAttribute("aria-selected", "true");
    expect(
      screen.getByRole("treeitem", { name: /Necessidade/ }),
    ).toHaveAttribute("aria-selected", "false");
  });

  it("mostra a obrigatoriedade de cada seção, em qualquer nível", () => {
    renderTree();

    const optional = screen.getByRole("treeitem", {
      name: /Desempenho mínimo/,
    });
    expect(optional).toHaveTextContent("Opcional");

    const required = screen.getByRole("treeitem", {
      name: /Requisitos técnicos/,
    });
    expect(required).toHaveTextContent("Obrig.");
  });
});
