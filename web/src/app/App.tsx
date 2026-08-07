import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";
import { BrowserRouter, Navigate, Route, Routes } from "react-router";

import { LoginPage } from "../features/auth/LoginPage";
import { DocumentsPage } from "../features/documents/DocumentsPage";
import { EditorWorkspace } from "../features/editor/EditorWorkspace";
import { TemplateEditorPage } from "../features/templates/TemplateEditorPage";
import { TemplatesPage } from "../features/templates/TemplatesPage";
import { RequireSession } from "./RequireSession";

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        // Document content is authoritative on the server; refetching on focus
        // would fight the local draft the person is editing.
        refetchOnWindowFocus: false,
        staleTime: 15_000,
      },
    },
  });
}

export function App() {
  const [queryClient] = useState(createQueryClient);

  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/entrar" element={<LoginPage />} />
          <Route element={<RequireSession />}>
            <Route path="/documentos" element={<DocumentsPage />} />
            <Route path="/documentos/:id" element={<EditorWorkspace />} />
            <Route path="/roteiros" element={<TemplatesPage />} />
            <Route path="/roteiros/:id" element={<TemplateEditorPage />} />
          </Route>
          <Route path="*" element={<Navigate to="/documentos" replace />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}
