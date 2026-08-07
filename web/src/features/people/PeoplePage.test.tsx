import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError, api, type Person } from "../../shared/api/client";
import { renderWithProviders } from "../../test/harness";
import { PeoplePage } from "./PeoplePage";

function people(): Person[] {
  return [
    {
      userId: "user-1",
      displayName: "Pierry A. Pereira",
      email: "pierry@orgao.gov.br",
      role: "organization_admin",
      pending: false,
    },
    {
      invitationId: "invite-1",
      displayName: "Marina Alves",
      email: "marina@orgao.gov.br",
      role: "author",
      pending: true,
      invitedUntil: "2026-08-10T12:00:00Z",
    },
  ];
}

function renderPeople() {
  return renderWithProviders(<PeoplePage />, { route: "/pessoas" });
}

describe("PeoplePage", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.spyOn(api, "session").mockResolvedValue({
      id: "user-1",
      displayName: "Pierry A. Pereira",
      role: "organization_admin",
    });
    vi.spyOn(api, "listPeople").mockResolvedValue(people());
  });

  it("mostra quem tem acesso e quem ainda tem convite pendente", async () => {
    renderPeople();

    expect(await screen.findByText("Pierry A. Pereira")).toBeVisible();
    expect(screen.getByText("Com acesso")).toBeVisible();
    expect(screen.getByText("Marina Alves")).toBeVisible();
    expect(screen.getByText("Convite pendente")).toBeVisible();
  });

  it("permite trocar o papel de quem já entrou", async () => {
    const user = userEvent.setup();
    const change = vi.spyOn(api, "changePersonRole").mockResolvedValue();
    renderPeople();
    await screen.findByText("Pierry A. Pereira");

    await user.selectOptions(
      screen.getByLabelText("Papel de Pierry A. Pereira"),
      "template_manager",
    );

    await waitFor(() => {
      expect(change).toHaveBeenCalledWith(
        "user-1",
        "organization_admin",
        "template_manager",
      );
    });
  });

  it("explica a recusa de remover a última administração", async () => {
    const user = userEvent.setup();
    vi.spyOn(api, "changePersonRole").mockRejectedValue(
      new ApiError(
        409,
        "A organização precisa manter ao menos uma administração",
      ),
    );
    renderPeople();
    await screen.findByText("Pierry A. Pereira");

    await user.selectOptions(
      screen.getByLabelText("Papel de Pierry A. Pereira"),
      "author",
    );

    expect(await screen.findByRole("alert")).toHaveTextContent(
      /ao menos uma administração/,
    );
  });

  it("revoga um convite pendente", async () => {
    const user = userEvent.setup();
    const revoke = vi.spyOn(api, "revokeInvitation").mockResolvedValue();
    renderPeople();
    await screen.findByText("Marina Alves");

    await user.click(screen.getByRole("button", { name: "Revogar convite" }));

    await waitFor(() => expect(revoke).toHaveBeenCalledWith("invite-1"));
  });

  // The token is shown once and cannot be retrieved: the screen has to make
  // that plain, or someone closes the dialog and loses the only copy.
  it("mostra o link do convite uma vez, com o aviso de que não volta", async () => {
    const user = userEvent.setup();
    vi.spyOn(api, "invitePerson").mockResolvedValue({
      inviteToken: "token-de-uso-unico",
      person: {
        invitationId: "invite-2",
        displayName: "Carlos Souza",
        email: "carlos@orgao.gov.br",
        role: "author",
        pending: true,
      },
    });
    renderPeople();
    await screen.findByText("Pierry A. Pereira");

    await user.click(screen.getByRole("button", { name: "Convidar pessoa" }));
    const dialog = await screen.findByRole("dialog");
    await user.type(within(dialog).getByLabelText("Nome"), "Carlos Souza");
    await user.type(
      within(dialog).getByLabelText("E-mail institucional"),
      "carlos@orgao.gov.br",
    );
    await user.click(
      within(dialog).getByRole("button", { name: "Criar convite" }),
    );

    const link =
      await screen.findByLabelText<HTMLInputElement>("Link do convite");
    expect(link.value).toContain("/convite?token=token-de-uso-unico");
    expect(
      screen.getByText(/serve uma única vez, vale por 72 horas/),
    ).toBeVisible();
  });

  it("relata quando o endereço já tem acesso", async () => {
    const user = userEvent.setup();
    vi.spyOn(api, "invitePerson").mockRejectedValue(
      new ApiError(409, "Este endereço já tem acesso"),
    );
    renderPeople();
    await screen.findByText("Pierry A. Pereira");

    await user.click(screen.getByRole("button", { name: "Convidar pessoa" }));
    const dialog = await screen.findByRole("dialog");
    await user.type(within(dialog).getByLabelText("Nome"), "Pierry");
    await user.type(
      within(dialog).getByLabelText("E-mail institucional"),
      "pierry@orgao.gov.br",
    );
    await user.click(
      within(dialog).getByRole("button", { name: "Criar convite" }),
    );

    expect(await screen.findByRole("alert")).toHaveTextContent(/já tem acesso/);
  });
});
