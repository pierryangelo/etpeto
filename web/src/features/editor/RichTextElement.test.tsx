import { render, screen, waitFor } from "@testing-library/react";
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
});

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
