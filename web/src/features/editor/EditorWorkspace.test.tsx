import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it } from "vitest";

import { EditorWorkspace } from "./EditorWorkspace";

describe("EditorWorkspace", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("recolhe o roteiro sem esconder o editor", async () => {
    const user = userEvent.setup();
    render(<EditorWorkspace />);

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
    render(<EditorWorkspace />);

    await user.click(
      screen.getByRole("button", { name: "Recolher prévia oficial" }),
    );

    expect(
      screen.queryByRole("complementary", { name: "Prévia oficial" }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole("navigation", { name: "Roteiro do documento" }),
    ).toBeVisible();
    expect(
      screen.getByRole("button", { name: "Expandir prévia oficial" }),
    ).toBeVisible();
  });

  it("restaura a preferência de painéis neste dispositivo", async () => {
    const user = userEvent.setup();
    const first = render(<EditorWorkspace />);
    await user.click(screen.getByRole("button", { name: "Recolher roteiro" }));
    first.unmount();

    render(<EditorWorkspace />);

    expect(
      screen.getByRole("button", { name: "Expandir roteiro" }),
    ).toBeVisible();
  });

  it("oferece modos exclusivos no viewport móvel", () => {
    render(<EditorWorkspace compact />);

    expect(screen.getByRole("tab", { name: "Editar" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    expect(screen.getByRole("tab", { name: "Roteiro" })).toBeVisible();
    expect(screen.getByRole("tab", { name: "Prévia" })).toBeVisible();
    expect(screen.getByRole("main")).toBeVisible();
  });

  it("expõe conteúdo real do primeiro ETP demonstrativo", () => {
    render(<EditorWorkspace />);

    expect(screen.getByText("Aquisição de notebooks")).toBeVisible();
    expect(
      screen.getByRole("heading", { name: "Requisitos da contratação" }),
    ).toBeVisible();
    expect(screen.getByText("Salvo agora")).toBeVisible();
  });
});
