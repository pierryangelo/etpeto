import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError, api } from "../../shared/api/client";
import { renderWithProviders, storedDocumentFixture } from "../../test/harness";
import { EditorWorkspace } from "./EditorWorkspace";

function renderWorkspace(props: { compact?: boolean } = {}) {
  return renderWithProviders(
    <EditorWorkspace documentId="documento-1" {...props} />,
    { route: "/documentos/documento-1", path: "/documentos/:id" },
  );
}

describe("EditorWorkspace", () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.restoreAllMocks();
    vi.spyOn(api, "openDocument").mockResolvedValue(storedDocumentFixture());
    vi.spyOn(api, "session").mockResolvedValue({
      id: "pessoa-1",
      displayName: "Pessoa Autora",
    });
  });

  it("mostra o roteiro e o título vindos do documento persistido", async () => {
    renderWorkspace();

    expect(
      await screen.findByRole("heading", {
        name: "Necessidade da contratação",
      }),
    ).toBeVisible();
    expect(screen.getByText("Aquisição de notebooks")).toBeVisible();
    expect(screen.getByText("2 seções")).toBeVisible();
    expect(screen.getByText("Revisão 1")).toBeVisible();
  });

  it("navega entre seções reais do roteiro", async () => {
    const user = userEvent.setup();
    renderWorkspace();
    await screen.findByRole("heading", { name: "Necessidade da contratação" });

    await user.click(
      screen.getByRole("button", { name: /Requisitos da contratação/ }),
    );

    expect(
      screen.getByRole("heading", { name: "Requisitos da contratação" }),
    ).toBeVisible();
    expect(screen.getByText("Seção 2 de 2")).toBeVisible();
  });

  it("recolhe o roteiro sem esconder o editor", async () => {
    const user = userEvent.setup();
    renderWorkspace();
    await screen.findByRole("heading", { name: "Necessidade da contratação" });

    await user.click(screen.getByRole("button", { name: "Recolher roteiro" }));

    expect(
      screen.queryByRole("navigation", { name: "Roteiro do documento" }),
    ).not.toBeInTheDocument();
    expect(screen.getByRole("main")).toBeVisible();
    expect(
      screen.getByRole("button", { name: "Expandir roteiro" }),
    ).toBeVisible();
  });

  it("recolhe a prévia de forma independente", async () => {
    const user = userEvent.setup();
    renderWorkspace();
    await screen.findByRole("heading", { name: "Necessidade da contratação" });

    await user.click(
      screen.getByRole("button", { name: "Recolher prévia oficial" }),
    );

    expect(
      screen.queryByRole("complementary", { name: "Prévia oficial" }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole("navigation", { name: "Roteiro do documento" }),
    ).toBeVisible();
  });

  it("restaura a preferência de painéis neste dispositivo", async () => {
    const user = userEvent.setup();
    const first = renderWorkspace();
    await screen.findByRole("heading", { name: "Necessidade da contratação" });
    await user.click(screen.getByRole("button", { name: "Recolher roteiro" }));
    first.unmount();

    renderWorkspace();

    expect(
      await screen.findByRole("button", { name: "Expandir roteiro" }),
    ).toBeVisible();
  });

  it("oferece modos exclusivos no viewport móvel", async () => {
    renderWorkspace({ compact: true });
    await screen.findByRole("heading", { name: "Necessidade da contratação" });

    expect(screen.getByRole("tab", { name: "Editar" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    expect(screen.getByRole("tab", { name: "Roteiro" })).toBeVisible();
    expect(screen.getByRole("tab", { name: "Prévia" })).toBeVisible();
  });

  it("redimensiona o roteiro e guarda a largura neste dispositivo", async () => {
    const user = userEvent.setup();
    const first = renderWorkspace();
    await screen.findByRole("heading", { name: "Necessidade da contratação" });

    const divider = screen.getByRole("separator", {
      name: "Redimensionar roteiro",
    });
    divider.focus();
    await user.keyboard("{ArrowRight}{ArrowRight}");

    expect(divider).toHaveAttribute("aria-valuenow", "264");
    first.unmount();

    renderWorkspace();
    expect(
      await screen.findByRole("separator", { name: "Redimensionar roteiro" }),
    ).toHaveAttribute("aria-valuenow", "264");
  });

  it("não deixa o roteiro encolher além do utilizável", async () => {
    const user = userEvent.setup();
    renderWorkspace();
    await screen.findByRole("heading", { name: "Necessidade da contratação" });

    const divider = screen.getByRole("separator", {
      name: "Redimensionar roteiro",
    });
    divider.focus();
    await user.keyboard("{Home}");

    expect(divider).toHaveAttribute("aria-valuenow", "200");
  });

  it("informa quando o documento não pode ser aberto", async () => {
    vi.spyOn(api, "openDocument").mockRejectedValue(
      new ApiError(404, "Documento não encontrado"),
    );
    renderWorkspace();

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent(
        /Não foi possível abrir este documento/,
      );
    });
  });
});
