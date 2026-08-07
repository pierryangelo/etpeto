import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router";

import { ApiError, api } from "../../shared/api/client";
import { AppHeader } from "../../shared/ui/AppHeader";
import { Button, PageHeading, TextField } from "../../shared/ui/controls";
import { DataTable } from "../../shared/ui/DataTable";
import { PageShell } from "../../shared/ui/PageShell";

export function DocumentsPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [title, setTitle] = useState("");
  const [error, setError] = useState("");

  const documents = useQuery({
    queryKey: ["documents"],
    queryFn: () => api.listDocuments(),
  });

  const create = useMutation({
    mutationFn: (nextTitle: string) => api.createDocument(nextTitle),
    onSuccess: async (created) => {
      setTitle("");
      await queryClient.invalidateQueries({ queryKey: ["documents"] });
      navigate(`/documentos/${created.id}`);
    },
    onError: (failure: unknown) => {
      setError(
        failure instanceof ApiError
          ? failure.message
          : "Não foi possível criar o documento.",
      );
    },
  });

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    const trimmed = title.trim();
    if (!trimmed) {
      setError("Informe um título para o documento.");
      return;
    }
    create.mutate(trimmed);
  }

  return (
    <PageShell header={<AppHeader />}>
      <PageHeading
        eyebrow="Estudos técnicos preliminares"
        title="Documentos da organização"
        description="Cada documento segue um roteiro institucional e gera o PDF oficial pelo mesmo pipeline usado na exportação."
      />

      <form className="flex items-end gap-3" onSubmit={submit} noValidate>
        <div className="flex-1">
          <TextField
            label="Título do novo ETP"
            name="title"
            value={title}
            maxLength={300}
            placeholder="Aquisição de notebooks"
            disabled={create.isPending}
            onChange={setTitle}
          />
        </div>
        <Button type="submit" variant="primary" disabled={create.isPending}>
          {create.isPending ? "Criando…" : "Criar ETP"}
        </Button>
      </form>

      {error && <p role="alert">{error}</p>}

      <section aria-labelledby="documents-title">
        <h2 id="documents-title" className="ds-label mb-3">
          Documentos existentes
        </h2>
        {documents.isPending && <p role="status">Carregando documentos…</p>}
        {documents.isError && (
          <p role="alert">
            Não foi possível carregar os documentos.{" "}
            <Button small onClick={() => void documents.refetch()}>
              Tentar novamente
            </Button>
          </p>
        )}
        {documents.isSuccess && (
          <DataTable
            columns={["Documento", "Revisão", "Atualizado"]}
            empty="Nenhum documento ainda. Crie o primeiro ETP acima."
            rows={documents.data.map((summary) => ({
              key: summary.id,
              cells: [
                {
                  emphasis: true,
                  content: (
                    <Link to={`/documentos/${summary.id}`}>
                      {summary.title}
                    </Link>
                  ),
                },
                { content: summary.revision, numeric: true },
                { content: formatUpdatedAt(summary.updatedAt) },
              ],
            }))}
          />
        )}
      </section>
    </PageShell>
  );
}

function formatUpdatedAt(value: string): string {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return "data desconhecida";
  return new Intl.DateTimeFormat("pt-BR", {
    dateStyle: "short",
    timeStyle: "short",
  }).format(parsed);
}
