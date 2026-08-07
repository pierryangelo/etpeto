import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { SerializedEditorState } from "lexical";
import { describe, expect, it, vi } from "vitest";

import { RichTextElement } from "./RichTextElement";

describe("RichTextElement", () => {
  it("preserva o estado Lexical no round-trip de edição", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <RichTextElement
        documentId="documento-1"
        elementId="requisitos-texto"
        initialState={paragraphState("Descreva as condições indispensáveis")}
        onChange={onChange}
      />,
    );

    const editor = screen.getByRole("textbox", { name: "Conteúdo da seção" });
    await user.click(editor);
    await user.keyboard(" e os níveis mínimos de serviço");

    await waitFor(() => {
      const lastCall = onChange.mock.lastCall?.[0] as
        SerializedEditorState | undefined;
      expect(JSON.stringify(lastCall)).toContain("níveis mínimos de serviço");
    });
  });

  it("expõe comandos de formatação com estado pressionado", () => {
    render(
      <RichTextElement
        documentId="documento-1"
        elementId="requisitos-texto"
        onChange={() => undefined}
      />,
    );

    expect(screen.getByRole("button", { name: "Negrito" })).toHaveAttribute(
      "aria-pressed",
      "false",
    );
    expect(screen.getByRole("button", { name: "Itálico" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Sublinhado" })).toBeVisible();
    expect(
      screen.getByRole("button", { name: "Lista com marcadores" }),
    ).toBeVisible();
    expect(screen.getByRole("button", { name: "Inserir link" })).toBeVisible();
  });

  // Disabled toolbar buttons stay focusable and are marked with aria-disabled,
  // so arrow navigation does not skip over them and change position under the
  // person's fingers.
  it("marca desfazer e refazer como indisponíveis enquanto não há o que reverter", () => {
    renderElement();

    expect(screen.getByRole("button", { name: "Desfazer" })).toHaveAttribute(
      "aria-disabled",
      "true",
    );
    expect(screen.getByRole("button", { name: "Refazer" })).toHaveAttribute(
      "aria-disabled",
      "true",
    );
  });

  it("habilita desfazer depois de escrever, e reverte o texto", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    renderElement({ onChange });

    const editor = screen.getByRole("textbox", { name: "Conteúdo da seção" });
    await user.click(editor);
    await user.keyboard("Texto que será revertido");

    const undo = screen.getByRole("button", { name: "Desfazer" });
    await waitFor(() => expect(undo).toHaveAttribute("aria-disabled", "false"));
    await user.click(undo);

    await waitFor(() => {
      expect(JSON.stringify(onChange.mock.lastCall?.[0])).not.toContain(
        "Texto que será revertido",
      );
    });
  });

  it("oferece os estilos de bloco em um menu, com o atual assinalado", async () => {
    const user = userEvent.setup();
    renderElement();

    await user.click(screen.getByRole("button", { name: "Estilo do bloco" }));

    const menu = await screen.findByRole("menu");
    expect(
      within(menu).getByRole("menuitem", { name: "Parágrafo" }),
    ).toBeVisible();
    expect(
      within(menu).getByRole("menuitem", { name: "Título" }),
    ).toBeVisible();
    expect(
      within(menu).getByRole("menuitem", { name: "Citação" }),
    ).toBeVisible();
  });

  it("aplica citação pelo menu de estilo do bloco", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    renderElement({
      initialState: paragraphState("Trecho citado da legislação"),
      onChange,
    });

    await user.click(
      screen.getByRole("textbox", { name: "Conteúdo da seção" }),
    );
    await user.click(screen.getByRole("button", { name: "Estilo do bloco" }));
    await user.click(await screen.findByRole("menuitem", { name: "Citação" }));

    await waitFor(() => {
      expect(JSON.stringify(onChange.mock.lastCall?.[0])).toContain('"quote"');
    });
  });

  it("oferece lista numerada além da lista com marcadores", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    renderElement({
      initialState: paragraphState("Primeiro requisito"),
      onChange,
    });

    await user.click(
      screen.getByRole("textbox", { name: "Conteúdo da seção" }),
    );
    await user.click(screen.getByRole("button", { name: "Lista numerada" }));

    await waitFor(() => {
      expect(JSON.stringify(onChange.mock.lastCall?.[0])).toContain('"number"');
    });
  });
});

function renderElement(
  props: Partial<React.ComponentProps<typeof RichTextElement>> = {},
) {
  return render(
    <RichTextElement
      documentId="documento-1"
      elementId="requisitos-texto"
      onChange={() => undefined}
      {...props}
    />,
  );
}

function paragraphState(text: string): SerializedEditorState {
  return {
    root: {
      children: [
        {
          children: [
            {
              detail: 0,
              format: 0,
              mode: "normal",
              style: "",
              text,
              type: "text",
              version: 1,
            },
          ],
          direction: null,
          format: "",
          indent: 0,
          textFormat: 0,
          textStyle: "",
          type: "paragraph",
          version: 1,
        },
      ],
      direction: null,
      format: "",
      indent: 0,
      type: "root",
      version: 1,
    },
  } as unknown as SerializedEditorState;
}
