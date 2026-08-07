import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, type RenderResult } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import type { ReactElement } from "react";

import type { StoredDocument } from "../shared/api/client";

/**
 * Renders a component inside the providers the application always supplies:
 * a router and a query client. Retries are disabled so a failing request
 * surfaces immediately instead of stalling the test.
 */
export function renderWithProviders(
  element: ReactElement,
  options: { route?: string; path?: string } = {},
): RenderResult {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  });
  const route = options.route ?? "/";
  const path = options.path ?? route;

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[route]}>
        <Routes>
          <Route path={path} element={element} />
          <Route path="*" element={<span>rota fora do teste</span>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

export function storedDocumentFixture(
  overrides: Partial<StoredDocument> = {},
): StoredDocument {
  return {
    id: "documento-1",
    revision: 1,
    updatedAt: "2026-08-07T12:00:00Z",
    content: {
      schemaVersion: 1,
      documentType: "etp",
      title: "Aquisição de notebooks",
      sections: [
        {
          id: "secao-necessidade",
          definitionKey: "necessidade",
          title: "Necessidade da contratação",
          guidance: "Descreva o problema a resolver.",
          elements: [{ id: "elemento-1", type: "richText" }],
        },
        {
          id: "secao-requisitos",
          definitionKey: "requisitos",
          title: "Requisitos da contratação",
          elements: [{ id: "elemento-2", type: "richText" }],
        },
      ],
    },
    ...overrides,
  };
}
