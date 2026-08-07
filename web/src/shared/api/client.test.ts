import { afterEach, describe, expect, it, vi } from "vitest";

import { ApiError, api, csrfCookieName, readCsrfToken } from "./client";

afterEach(() => {
  vi.unstubAllGlobals();
  document.cookie = `${csrfCookieName}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/; secure`;
});

function stubFetch(response: Response) {
  const fetchMock = vi.fn().mockResolvedValue(response);
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("readCsrfToken", () => {
  it("lê o token publicado no cookie legível", () => {
    document.cookie = `${csrfCookieName}=token-de-teste; path=/; secure`;
    expect(readCsrfToken()).toBe("token-de-teste");
  });

  it("devolve vazio quando o cookie não existe", () => {
    expect(readCsrfToken()).toBe("");
  });
});

describe("api", () => {
  it("envia o token CSRF em requisições mutáveis", async () => {
    document.cookie = `${csrfCookieName}=token-de-teste; path=/; secure`;
    const fetchMock = stubFetch(
      jsonResponse({
        id: "documento-1",
        revision: 1,
        updatedAt: "2026-08-07T00:00:00Z",
        content: {
          schemaVersion: 1,
          documentType: "etp",
          title: "ETP",
          sections: [],
        },
      }),
    );

    await api.createDocument("ETP");

    const init = requestInit(fetchMock);
    expect(init.method).toBe("POST");
    expect(init.headers["X-Etpeto-Csrf"]).toBe("token-de-teste");
    expect(init.credentials).toBe("include");
  });

  it("não envia o token CSRF em leituras", async () => {
    document.cookie = `${csrfCookieName}=token-de-teste; path=/; secure`;
    const fetchMock = stubFetch(jsonResponse({ documents: [] }));

    await api.listDocuments();

    const init = requestInit(fetchMock);
    expect(init.headers["X-Etpeto-Csrf"]).toBeUndefined();
  });

  it("expõe conflito de revisão para o chamador tratar", async () => {
    stubFetch(
      jsonResponse(
        { detail: "O documento foi alterado por outra sessão" },
        409,
      ),
    );

    const failure = await api
      .saveDocument("documento-1", 1, {
        schemaVersion: 1,
        documentType: "etp",
        title: "ETP",
        sections: [],
      })
      .catch((error: unknown) => error);

    expect(failure).toBeInstanceOf(ApiError);
    expect((failure as ApiError).isConflict).toBe(true);
    expect((failure as ApiError).message).toBe(
      "O documento foi alterado por outra sessão",
    );
  });

  it("identifica a sessão ausente para a interface redirecionar", async () => {
    stubFetch(jsonResponse({ detail: "Sessão inválida ou expirada" }, 401));

    const failure = await api.session().catch((error: unknown) => error);

    expect(failure).toBeInstanceOf(ApiError);
    expect((failure as ApiError).isUnauthenticated).toBe(true);
  });

  it("recusa uma resposta fora do contrato em vez de propagar dado inválido", async () => {
    stubFetch(jsonResponse({ documents: [{ id: "documento-1" }] }));

    await expect(api.listDocuments()).rejects.toThrow();
  });
});

function requestInit(fetchMock: ReturnType<typeof vi.fn>): RequestInit & {
  headers: Record<string, string>;
} {
  const call = fetchMock.mock.calls[0];
  if (!call) throw new Error("fetch was not called");
  return call[1] as RequestInit & { headers: Record<string, string> };
}
