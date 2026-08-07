import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { SerializedEditorState } from "lexical";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError, api } from "../../shared/api/client";
import { storedDocumentFixture } from "../../test/harness";
import { useDocumentEditor } from "./useDocumentEditor";

function wrapper({ children }: { children: ReactNode }) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  });
  return (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

function lexicalState(text: string): SerializedEditorState {
  return {
    root: {
      type: "root",
      version: 1,
      children: [
        {
          type: "paragraph",
          version: 1,
          children: [{ type: "text", version: 1, text }],
        },
      ],
    },
  } as unknown as SerializedEditorState;
}

async function loadedEditor() {
  const view = renderHook(() => useDocumentEditor("documento-1"), { wrapper });
  await waitFor(() => expect(view.result.current.draft).not.toBeNull());
  return view;
}

describe("useDocumentEditor", () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    vi.spyOn(api, "openDocument").mockResolvedValue(storedDocumentFixture());
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("agrupa digitação seguida em um único salvamento", async () => {
    const save = vi
      .spyOn(api, "saveDocument")
      .mockResolvedValue(storedDocumentFixture({ revision: 2 }));
    const view = await loadedEditor();

    act(() => {
      view.result.current.updateElement(
        "secao-necessidade",
        "elemento-1",
        lexicalState("primeiro"),
      );
    });
    act(() => {
      view.result.current.updateElement(
        "secao-necessidade",
        "elemento-1",
        lexicalState("primeiro e segundo"),
      );
    });
    expect(view.result.current.saveState).toBe("dirty");
    expect(save).not.toHaveBeenCalled();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1500);
    });

    await waitFor(() => expect(save).toHaveBeenCalledTimes(1));
    const [, expectedRevision, content] = save.mock.calls[0]!;
    expect(expectedRevision).toBe(1);
    expect(content.sections[0]!.elements[0]!.richText).toEqual(
      lexicalState("primeiro e segundo"),
    );
  });

  it("envia o pendente imediatamente quando a prévia é solicitada", async () => {
    const save = vi
      .spyOn(api, "saveDocument")
      .mockResolvedValue(storedDocumentFixture({ revision: 2 }));
    const view = await loadedEditor();

    act(() => {
      view.result.current.updateElement(
        "secao-necessidade",
        "elemento-1",
        lexicalState("texto"),
      );
    });
    act(() => {
      view.result.current.flush();
    });

    await waitFor(() => expect(save).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(view.result.current.saveState).toBe("saved"));
  });

  it("usa a revisão devolvida pelo servidor no salvamento seguinte", async () => {
    const save = vi
      .spyOn(api, "saveDocument")
      .mockResolvedValueOnce(storedDocumentFixture({ revision: 2 }))
      .mockResolvedValueOnce(storedDocumentFixture({ revision: 3 }));
    const view = await loadedEditor();

    act(() => {
      view.result.current.updateElement(
        "secao-necessidade",
        "elemento-1",
        lexicalState("primeiro"),
      );
      view.result.current.flush();
    });
    await waitFor(() => expect(view.result.current.saveState).toBe("saved"));

    act(() => {
      view.result.current.updateElement(
        "secao-necessidade",
        "elemento-1",
        lexicalState("segundo"),
      );
      view.result.current.flush();
    });

    await waitFor(() => expect(save).toHaveBeenCalledTimes(2));
    expect(save.mock.calls[1]![1]).toBe(2);
  });

  it("relata conflito sem descartar o texto local", async () => {
    vi.spyOn(api, "saveDocument").mockRejectedValue(
      new ApiError(409, "O documento foi alterado por outra sessão"),
    );
    const view = await loadedEditor();

    act(() => {
      view.result.current.updateElement(
        "secao-necessidade",
        "elemento-1",
        lexicalState("texto que não pode sumir"),
      );
      view.result.current.flush();
    });

    await waitFor(() => expect(view.result.current.saveState).toBe("conflict"));
    expect(
      view.result.current.draft?.sections[0]?.elements[0]?.richText,
    ).toEqual(lexicalState("texto que não pode sumir"));
  });

  it("distingue falha de rede de conflito", async () => {
    vi.spyOn(api, "saveDocument").mockRejectedValue(
      new ApiError(503, "Documentos temporariamente indisponíveis"),
    );
    const view = await loadedEditor();

    act(() => {
      view.result.current.updateElement(
        "secao-necessidade",
        "elemento-1",
        lexicalState("texto"),
      );
      view.result.current.flush();
    });

    await waitFor(() => expect(view.result.current.saveState).toBe("failed"));
  });

  it("descarta o rascunho local ao recarregar a revisão remota", async () => {
    const remote = storedDocumentFixture({ revision: 5 });
    remote.content.title = "Título vindo do servidor";
    vi.spyOn(api, "saveDocument").mockRejectedValue(
      new ApiError(409, "conflito"),
    );
    const view = await loadedEditor();

    act(() => {
      view.result.current.updateElement(
        "secao-necessidade",
        "elemento-1",
        lexicalState("texto local"),
      );
      view.result.current.flush();
    });
    await waitFor(() => expect(view.result.current.saveState).toBe("conflict"));

    vi.spyOn(api, "openDocument").mockResolvedValue(remote);
    await act(async () => {
      await view.result.current.reloadRemote();
    });

    expect(view.result.current.draft?.title).toBe("Título vindo do servidor");
    expect(view.result.current.saveState).toBe("saved");
  });

  it("reenvia o texto local sobre a revisão atual quando o autor decide mantê-lo", async () => {
    const save = vi
      .spyOn(api, "saveDocument")
      .mockRejectedValueOnce(new ApiError(409, "conflito"))
      .mockResolvedValueOnce(storedDocumentFixture({ revision: 6 }));
    const view = await loadedEditor();

    act(() => {
      view.result.current.updateElement(
        "secao-necessidade",
        "elemento-1",
        lexicalState("texto local"),
      );
      view.result.current.flush();
    });
    await waitFor(() => expect(view.result.current.saveState).toBe("conflict"));

    vi.spyOn(api, "openDocument").mockResolvedValue(
      storedDocumentFixture({ revision: 5 }),
    );
    await act(async () => {
      await view.result.current.keepLocal();
    });

    await waitFor(() => expect(save).toHaveBeenCalledTimes(2));
    expect(save.mock.calls[1]![1]).toBe(5);
    expect(save.mock.calls[1]![2].sections[0]!.elements[0]!.richText).toEqual(
      lexicalState("texto local"),
    );
  });

  it("não perde uma alteração pendente quando o editor é desmontado", async () => {
    const save = vi
      .spyOn(api, "saveDocument")
      .mockResolvedValue(storedDocumentFixture({ revision: 2 }));
    const view = await loadedEditor();

    act(() => {
      view.result.current.updateElement(
        "secao-necessidade",
        "elemento-1",
        lexicalState("digitado antes de sair"),
      );
    });
    view.unmount();

    await waitFor(() => expect(save).toHaveBeenCalledTimes(1));
  });
});
