import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError, api } from "../../shared/api/client";
import { renderWithProviders, storedDocumentFixture } from "../../test/harness";
import { DocumentsPage } from "./DocumentsPage";

describe("DocumentsPage", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.spyOn(api, "session").mockResolvedValue({
      id: "pessoa-1",
      displayName: "Pessoa Autora",
      role: "author",
    });
  });

  it("lista os documentos da organização", async () => {
    vi.spyOn(api, "listDocuments").mockResolvedValue([
      {
        id: "documento-1",
        title: "Aquisição de notebooks",
        revision: 3,
        updatedAt: "2026-08-07T12:00:00Z",
      },
    ]);

    renderWithProviders(<DocumentsPage />, { route: "/documentos" });

    const link = await screen.findByRole("link", {
      name: /Aquisição de notebooks/,
    });
    expect(link).toHaveAttribute("href", "/documentos/documento-1");
    // The revision has its own column now, so it reads as a number.
    expect(screen.getByRole("cell", { name: "3" })).toBeVisible();
  });

  it("orienta quando ainda não existe documento", async () => {
    vi.spyOn(api, "listDocuments").mockResolvedValue([]);

    renderWithProviders(<DocumentsPage />, { route: "/documentos" });

    expect(await screen.findByText(/Nenhum documento ainda/)).toBeVisible();
  });

  it("cria um ETP e abre o documento criado", async () => {
    const user = userEvent.setup();
    vi.spyOn(api, "listDocuments").mockResolvedValue([]);
    const create = vi
      .spyOn(api, "createDocument")
      .mockResolvedValue(storedDocumentFixture());

    renderWithProviders(<DocumentsPage />, { route: "/documentos" });
    await screen.findByText(/Nenhum documento ainda/);

    await user.type(
      screen.getByRole("textbox", { name: /Título do novo ETP/ }),
      "Aquisição de notebooks",
    );
    await user.click(screen.getByRole("button", { name: "Criar ETP" }));

    await waitFor(() => {
      expect(create).toHaveBeenCalledWith("Aquisição de notebooks");
    });
  });

  it("recusa a criação sem título antes de chamar a API", async () => {
    const user = userEvent.setup();
    vi.spyOn(api, "listDocuments").mockResolvedValue([]);
    const create = vi.spyOn(api, "createDocument");

    renderWithProviders(<DocumentsPage />, { route: "/documentos" });
    await screen.findByText(/Nenhum documento ainda/);

    await user.click(screen.getByRole("button", { name: "Criar ETP" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      /Informe um título/,
    );
    expect(create).not.toHaveBeenCalled();
  });

  it("permite tentar novamente quando a listagem falha", async () => {
    vi.spyOn(api, "listDocuments").mockRejectedValue(
      new ApiError(503, "Documentos temporariamente indisponíveis"),
    );

    renderWithProviders(<DocumentsPage />, { route: "/documentos" });

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent(
        /Não foi possível carregar os documentos/,
      );
    });
    expect(
      screen.getByRole("button", { name: "Tentar novamente" }),
    ).toBeVisible();
  });
});
