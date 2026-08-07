import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { PanelDivider } from "./PanelDivider";

function renderDivider(
  overrides: Partial<Parameters<typeof PanelDivider>[0]> = {},
) {
  const onResize = vi.fn();
  const onReset = vi.fn();
  render(
    <PanelDivider
      label="Redimensionar roteiro"
      width={240}
      min={200}
      max={420}
      grow="right"
      onResize={onResize}
      onReset={onReset}
      {...overrides}
    />,
  );
  return { onResize, onReset };
}

describe("PanelDivider", () => {
  it("anuncia a largura e os limites do painel", () => {
    renderDivider();

    const divider = screen.getByRole("separator", {
      name: "Redimensionar roteiro",
    });
    expect(divider).toHaveAttribute("aria-valuenow", "240");
    expect(divider).toHaveAttribute("aria-valuemin", "200");
    expect(divider).toHaveAttribute("aria-valuemax", "420");
  });

  it("redimensiona pelo teclado, para quem não usa arraste", async () => {
    const user = userEvent.setup();
    const { onResize } = renderDivider();

    const divider = screen.getByRole("separator");
    divider.focus();
    await user.keyboard("{ArrowRight}");
    expect(onResize).toHaveBeenLastCalledWith(256);

    await user.keyboard("{ArrowLeft}");
    expect(onResize).toHaveBeenLastCalledWith(224);

    await user.keyboard("{Home}");
    expect(onResize).toHaveBeenLastCalledWith(200);

    await user.keyboard("{End}");
    expect(onResize).toHaveBeenLastCalledWith(420);
  });

  it("inverte o sentido no painel que cresce para a esquerda", async () => {
    const user = userEvent.setup();
    const { onResize } = renderDivider({ grow: "left" });

    screen.getByRole("separator").focus();
    await user.keyboard("{ArrowLeft}");

    expect(onResize).toHaveBeenLastCalledWith(256);
  });

  it("volta à largura padrão com Enter ou duplo clique", async () => {
    const user = userEvent.setup();
    const { onReset } = renderDivider();

    const divider = screen.getByRole("separator");
    divider.focus();
    await user.keyboard("{Enter}");
    await user.dblClick(divider);

    expect(onReset).toHaveBeenCalledTimes(2);
  });

  it("é alcançável pelo teclado", () => {
    renderDivider();
    expect(screen.getByRole("separator")).toHaveAttribute("tabindex", "0");
  });

  it("acompanha o arraste do ponteiro a partir da largura inicial", () => {
    const { onResize } = renderDivider();
    const divider = screen.getByRole("separator");

    fireEvent.pointerDown(divider, { button: 0, pointerId: 1, clientX: 300 });
    fireEvent.pointerMove(divider, { pointerId: 1, clientX: 360 });
    expect(onResize).toHaveBeenLastCalledWith(300);

    fireEvent.pointerMove(divider, { pointerId: 1, clientX: 280 });
    expect(onResize).toHaveBeenLastCalledWith(220);

    fireEvent.pointerUp(divider, { pointerId: 1 });
    onResize.mockClear();
    fireEvent.pointerMove(divider, { pointerId: 1, clientX: 500 });
    expect(onResize).not.toHaveBeenCalled();
  });

  it("ignora movimento sem arraste iniciado", () => {
    const { onResize } = renderDivider();

    fireEvent.pointerMove(screen.getByRole("separator"), {
      pointerId: 1,
      clientX: 500,
    });

    expect(onResize).not.toHaveBeenCalled();
  });
});
