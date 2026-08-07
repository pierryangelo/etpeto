import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError, api } from "../../shared/api/client";
import { renderWithProviders } from "../../test/harness";
import { AcceptInvitePage } from "./AcceptInvitePage";

function renderInvite(token = "token-valido") {
  return renderWithProviders(<AcceptInvitePage />, {
    route: `/convite?token=${token}`,
    path: "/convite",
  });
}

describe("AcceptInvitePage", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.spyOn(api, "describeInvitation").mockResolvedValue({
      displayName: "Marina Alves",
      email: "marina@orgao.gov.br",
    });
  });

  it("recebe a pessoa pelo nome que a administração escreveu", async () => {
    renderInvite();

    expect(await screen.findByText(/Bem-vindo, Marina Alves/)).toBeVisible();
    expect(screen.getByText(/marina@orgao.gov.br/)).toBeVisible();
  });

  it("define a senha e leva para a entrada", async () => {
    const user = userEvent.setup();
    const accept = vi.spyOn(api, "acceptInvitation").mockResolvedValue();
    renderInvite();
    await screen.findByText(/Bem-vindo/);

    await user.type(screen.getByLabelText("Senha"), "senha escolhida por mim");
    await user.type(
      screen.getByLabelText("Repita a senha"),
      "senha escolhida por mim",
    );
    await user.click(
      screen.getByRole("button", { name: "Definir senha e entrar" }),
    );

    await waitFor(() => {
      expect(accept).toHaveBeenCalledWith(
        "token-valido",
        "senha escolhida por mim",
      );
    });
  });

  it("recusa senhas que não coincidem antes de chamar a API", async () => {
    const user = userEvent.setup();
    const accept = vi.spyOn(api, "acceptInvitation");
    renderInvite();
    await screen.findByText(/Bem-vindo/);

    await user.type(screen.getByLabelText("Senha"), "senha escolhida por mim");
    await user.type(
      screen.getByLabelText("Repita a senha"),
      "outra senha aqui",
    );
    await user.click(
      screen.getByRole("button", { name: "Definir senha e entrar" }),
    );

    expect(await screen.findByRole("alert")).toHaveTextContent(/não coincidem/);
    expect(accept).not.toHaveBeenCalled();
  });

  it("cobra o piso de tamanho da senha antes de chamar a API", async () => {
    const user = userEvent.setup();
    const accept = vi.spyOn(api, "acceptInvitation");
    renderInvite();
    await screen.findByText(/Bem-vindo/);

    await user.type(screen.getByLabelText("Senha"), "curta");
    await user.type(screen.getByLabelText("Repita a senha"), "curta");
    await user.click(
      screen.getByRole("button", { name: "Definir senha e entrar" }),
    );

    expect(await screen.findByRole("alert")).toHaveTextContent(
      /ao menos 12 caracteres/,
    );
    expect(accept).not.toHaveBeenCalled();
  });

  // A used or expired link must say so plainly, and point at the way out.
  it("explica um convite que não pode mais ser usado", async () => {
    vi.spyOn(api, "describeInvitation").mockRejectedValue(
      new ApiError(410, "Este convite expirou"),
    );
    renderInvite();

    expect(
      await screen.findByText(/Este convite não pode ser usado/),
    ).toBeVisible();
    expect(screen.getByText(/Peça um novo à administração/)).toBeVisible();
  });

  it("trata um link sem token", async () => {
    renderWithProviders(<AcceptInvitePage />, {
      route: "/convite",
      path: "/convite",
    });

    expect(
      await screen.findByText(/Este convite não pode ser usado/),
    ).toBeVisible();
  });
});
