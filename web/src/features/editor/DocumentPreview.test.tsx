import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError, api } from "../../shared/api/client";
import { renderWithProviders } from "../../test/harness";
import { DocumentPreview } from "./DocumentPreview";

const objectUrl = "blob:https://localhost/preview-1";

describe("DocumentPreview", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.stubGlobal("URL", {
      ...URL,
      createObjectURL: vi.fn().mockReturnValue(objectUrl),
      revokeObjectURL: vi.fn(),
    });
  });

  it("só gera o PDF quando solicitado, preservando a edição", () => {
    const preview = vi.spyOn(api, "previewDocument");
    renderWithProviders(
      <DocumentPreview documentId="documento-1" revision={1} />,
    );

    expect(preview).not.toHaveBeenCalled();
    expect(screen.getByText(/Gere a prévia para ver o PDF/)).toBeVisible();
  });

  it("envia o salvamento pendente antes de renderizar", async () => {
    const user = userEvent.setup();
    const flush = vi.fn();
    vi.spyOn(api, "previewDocument").mockResolvedValue(new Blob(["%PDF"]));

    renderWithProviders(
      <DocumentPreview
        documentId="documento-1"
        revision={1}
        onBeforeRender={flush}
      />,
    );
    await user.click(screen.getByRole("button", { name: "Atualizar" }));

    expect(flush).toHaveBeenCalled();
    await waitFor(() => {
      expect(screen.getByTitle("Prévia oficial em PDF")).toHaveAttribute(
        "src",
        objectUrl,
      );
    });
  });

  it("libera a URL anterior ao gerar uma nova prévia", async () => {
    const user = userEvent.setup();
    vi.spyOn(api, "previewDocument").mockResolvedValue(new Blob(["%PDF"]));

    renderWithProviders(
      <DocumentPreview documentId="documento-1" revision={1} />,
    );
    await user.click(screen.getByRole("button", { name: "Atualizar" }));
    await screen.findByTitle("Prévia oficial em PDF");
    await user.click(screen.getByRole("button", { name: "Atualizar" }));

    await waitFor(() => {
      expect(URL.revokeObjectURL).toHaveBeenCalledWith(objectUrl);
    });
  });

  it("mantém a edição utilizável quando o renderer falha", async () => {
    const user = userEvent.setup();
    vi.spyOn(api, "previewDocument").mockRejectedValue(
      new ApiError(503, "Renderização temporariamente indisponível"),
    );

    renderWithProviders(
      <DocumentPreview documentId="documento-1" revision={1} />,
    );
    await user.click(screen.getByRole("button", { name: "Atualizar" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      /A renderização falhou/,
    );
    expect(screen.getByRole("button", { name: "Atualizar" })).toBeEnabled();
  });

  it("avisa quando o PDF está atrás da revisão atual", async () => {
    const user = userEvent.setup();
    vi.spyOn(api, "previewDocument").mockResolvedValue(new Blob(["%PDF"]));

    function Harness() {
      const [revision, setRevision] = useState(2);
      return (
        <>
          <button type="button" onClick={() => setRevision(3)}>
            Salvar nova revisão
          </button>
          <DocumentPreview documentId="documento-1" revision={revision} />
        </>
      );
    }

    renderWithProviders(<Harness />);
    await user.click(screen.getByRole("button", { name: "Atualizar" }));
    await screen.findByTitle("Prévia oficial em PDF");

    await user.click(
      screen.getByRole("button", { name: "Salvar nova revisão" }),
    );

    expect(await screen.findByRole("status")).toHaveTextContent(
      /revisão 2.*está na 3/s,
    );
  });

  it("só habilita a exportação depois de uma renderização válida", async () => {
    const user = userEvent.setup();
    vi.spyOn(api, "previewDocument").mockResolvedValue(new Blob(["%PDF"]));

    renderWithProviders(
      <DocumentPreview documentId="documento-1" revision={1} />,
    );
    expect(screen.getByRole("button", { name: "Exportar PDF" })).toBeDisabled();

    await user.click(screen.getByRole("button", { name: "Atualizar" }));

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Exportar PDF" }),
      ).toBeEnabled();
    });
  });

  it("baixa o PDF já renderizado com um nome de arquivo seguro", async () => {
    const user = userEvent.setup();
    vi.spyOn(api, "previewDocument").mockResolvedValue(new Blob(["%PDF"]));
    const click = vi
      .spyOn(HTMLAnchorElement.prototype, "click")
      .mockImplementation(() => undefined);

    renderWithProviders(
      <DocumentPreview
        documentId="documento-1"
        documentTitle="Aquisição de notebooks"
        revision={1}
      />,
    );
    await user.click(screen.getByRole("button", { name: "Atualizar" }));
    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Exportar PDF" }),
      ).toBeEnabled();
    });
    await user.click(screen.getByRole("button", { name: "Exportar PDF" }));

    expect(click).toHaveBeenCalled();
    const anchor = click.mock.instances[0] as HTMLAnchorElement;
    expect(anchor.download).toBe("aquisicao-de-notebooks.pdf");
    expect(anchor.href).toBe(objectUrl);
  });
});
