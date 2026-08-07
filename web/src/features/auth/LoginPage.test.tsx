import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { LoginPage } from "./LoginPage";

describe("LoginPage", () => {
  it("confirma a sessão antes de navegar e bloqueia envio duplicado", async () => {
    const user = userEvent.setup();
    let finishLogin: (() => void) | undefined;
    const login = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          finishLogin = resolve;
        }),
    );
    const confirmSession = vi.fn().mockResolvedValue(undefined);
    const navigate = vi.fn();
    render(
      <LoginPage
        login={login}
        confirmSession={confirmSession}
        navigate={navigate}
      />,
    );

    await user.type(
      screen.getByRole("textbox", { name: "E-mail institucional" }),
      "pessoa@orgao.gov.br",
    );
    const password = screen.getByLabelText("Senha");
    await user.type(password, "senha segura{Enter}");

    expect(login).toHaveBeenCalledOnce();
    expect(screen.getByRole("button", { name: "Entrando…" })).toBeDisabled();
    await user.keyboard("{Enter}");
    expect(login).toHaveBeenCalledOnce();

    finishLogin?.();
    await waitFor(() => expect(confirmSession).toHaveBeenCalledOnce());
    expect(navigate).toHaveBeenCalledWith("/documentos");
  });

  it("anuncia falhas de autenticação", async () => {
    const user = userEvent.setup();
    render(
      <LoginPage
        login={() => Promise.reject(new Error("unauthorized"))}
        confirmSession={() => Promise.resolve()}
        navigate={() => undefined}
      />,
    );

    await user.type(
      screen.getByRole("textbox", { name: "E-mail institucional" }),
      "pessoa@orgao.gov.br",
    );
    await user.type(screen.getByLabelText("Senha"), "incorreta");
    await user.click(screen.getByRole("button", { name: "Entrar" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Não foi possível entrar",
    );
  });
});
