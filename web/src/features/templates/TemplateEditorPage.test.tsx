import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError, api, type TemplateDraft } from "../../shared/api/client";
import { renderWithProviders } from "../../test/harness";
import { TemplateEditorPage } from "./TemplateEditorPage";

function draftFixture(overrides: Partial<TemplateDraft> = {}): TemplateDraft {
  return {
    id: "template-1",
    documentType: "etp",
    name: "Estudo Técnico Preliminar",
    draftVersion: 3,
    draftUpdatedAt: "2026-08-07T12:00:00Z",
    publishedRevision: 3,
    blueprint: {
      schemaVersion: 1,
      documentType: "etp",
      sections: [
        {
          key: "necessidade",
          title: "Necessidade da contratação",
          guidance: "Descreva o problema a resolver.",
          required: true,
          allowRename: false,
          allowedElements: ["richText"],
        },
        {
          key: "requisitos",
          title: "Requisitos da contratação",
          required: true,
          allowRename: false,
          allowedElements: ["richText"],
          sections: [
            {
              key: "tecnicos",
              title: "Requisitos técnicos",
              required: false,
              allowRename: false,
              allowedElements: ["richText"],
            },
          ],
        },
      ],
    },
    ...overrides,
  };
}

function renderEditor() {
  return renderWithProviders(<TemplateEditorPage />, {
    route: "/roteiros/template-1",
    path: "/roteiros/:id",
  });
}

describe("TemplateEditorPage", () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    vi.restoreAllMocks();
    vi.spyOn(api, "openTemplate").mockResolvedValue(draftFixture());
    vi.spyOn(api, "session").mockResolvedValue({
      id: "pessoa-1",
      displayName: "Pessoa Gestora",
      role: "template_manager",
    });
  });

  it("mostra a estrutura numerada com a obrigatoriedade de cada seção", async () => {
    renderEditor();

    expect(await screen.findByText("2.1")).toBeVisible();
    const structure = screen.getByRole("navigation", {
      name: "Estrutura do roteiro",
    });
    expect(within(structure).getByText("Requisitos técnicos")).toBeVisible();
    expect(within(structure).getAllByText("Obrig.")).toHaveLength(2);
    expect(within(structure).getByText("Opcional")).toBeVisible();
  });

  // The manager has to know what the authors are still using while editing.
  it("informa em qual revisão os autores estão", async () => {
    renderEditor();

    expect(
      await screen.findByText(/os autores continuam na revisão 3/),
    ).toBeVisible();
    expect(
      screen.getByRole("button", { name: "Publicar revisão 4" }),
    ).toBeVisible();
  });

  it("edita o título da seção e salva o rascunho", async () => {
    const user = userEvent.setup();
    const save = vi
      .spyOn(api, "saveTemplateDraft")
      .mockResolvedValue(draftFixture({ draftVersion: 4 }));
    renderEditor();
    await screen.findByText("2.1");

    await user.type(screen.getByLabelText(/Título/), " revisado");
    await vi.advanceTimersByTimeAsync(1500);

    await waitFor(() => expect(save).toHaveBeenCalled());
    const [, expectedVersion, blueprint] = save.mock.lastCall!;
    expect(expectedVersion).toBe(3);
    expect(blueprint.sections[0]!.title).toContain("revisado");
    // Renaming never changes the key a document carries.
    expect(blueprint.sections[0]!.key).toBe("necessidade");
  });

  it("escreve a orientação que o autor vai ler", async () => {
    const user = userEvent.setup();
    const save = vi
      .spyOn(api, "saveTemplateDraft")
      .mockResolvedValue(draftFixture({ draftVersion: 4 }));
    renderEditor();
    await screen.findByText("2.1");

    await user.clear(screen.getByLabelText(/Orientação/));
    await user.type(
      screen.getByLabelText(/Orientação/),
      "Explique a motivação.",
    );
    await vi.advanceTimersByTimeAsync(1500);

    await waitFor(() => expect(save).toHaveBeenCalled());
    expect(save.mock.lastCall![2].sections[0]!.guidance).toBe(
      "Explique a motivação.",
    );
  });

  it("alterna a obrigatoriedade da seção", async () => {
    const user = userEvent.setup();
    const save = vi
      .spyOn(api, "saveTemplateDraft")
      .mockResolvedValue(draftFixture({ draftVersion: 4 }));
    renderEditor();
    await screen.findByText("2.1");

    await user.click(screen.getByRole("switch", { name: "Seção obrigatória" }));
    await vi.advanceTimersByTimeAsync(1500);

    await waitFor(() => expect(save).toHaveBeenCalled());
    expect(save.mock.lastCall![2].sections[0]!.required).toBe(false);
  });

  it("acrescenta uma subseção à seção selecionada", async () => {
    const user = userEvent.setup();
    const save = vi
      .spyOn(api, "saveTemplateDraft")
      .mockResolvedValue(draftFixture({ draftVersion: 4 }));
    renderEditor();
    await screen.findByText("2.1");

    await user.click(screen.getByRole("button", { name: "＋ Subseção" }));
    await vi.advanceTimersByTimeAsync(1500);

    await waitFor(() => expect(save).toHaveBeenCalled());
    const added = save.mock.lastCall![2].sections[0]!.sections;
    expect(added).toHaveLength(1);
    expect(added![0]!.key).toBe("nova-subsecao");
  });

  it("desabilita promover quando a seção já está na raiz", async () => {
    renderEditor();
    await screen.findByText("2.1");

    expect(screen.getByRole("button", { name: "← Promover" })).toBeDisabled();
  });

  it("relata conflito quando outra sessão salvou o rascunho", async () => {
    const user = userEvent.setup();
    vi.spyOn(api, "saveTemplateDraft").mockRejectedValue(
      new ApiError(409, "O rascunho foi alterado em outra sessão"),
    );
    renderEditor();
    await screen.findByText("2.1");

    await user.type(screen.getByLabelText(/Título/), "x");
    await vi.advanceTimersByTimeAsync(1500);

    await waitFor(() => {
      expect(screen.getByText("Conflito detectado")).toBeVisible();
    });
  });

  it("explica a recusa de publicar um rascunho sem alterações", async () => {
    const user = userEvent.setup();
    vi.spyOn(api, "publishTemplate").mockRejectedValue(
      new ApiError(422, "O rascunho é igual à revisão publicada"),
    );
    renderEditor();
    await screen.findByText("2.1");

    await user.click(
      screen.getByRole("button", { name: "Publicar revisão 4" }),
    );

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent(
        /Não foi possível publicar/,
      );
    });
  });

  it("avisa que remover não afeta documentos já criados", async () => {
    renderEditor();
    await screen.findByText("2.1");

    expect(
      screen.getByText(/Remover não afeta documentos já criados/),
    ).toBeVisible();
  });
});
