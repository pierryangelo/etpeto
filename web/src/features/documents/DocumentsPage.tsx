import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router";

import { ApiError, api } from "../../shared/api/client";
import { AppHeader } from "../../shared/ui/AppHeader";
import styles from "./DocumentsPage.module.css";

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
    <div className={styles.page}>
      <AppHeader />
      <main className={styles.content}>
        <div className={styles.intro}>
          <span className={styles.eyebrow}>Estudos técnicos preliminares</span>
          <h1>Documentos da organização</h1>
          <p>
            Cada documento segue um roteiro institucional e gera o PDF oficial
            pelo mesmo pipeline usado na exportação.
          </p>
        </div>

        <form className={styles.createCard} onSubmit={submit} noValidate>
          <label className={styles.field}>
            <span>Título do novo ETP</span>
            <input
              name="title"
              value={title}
              maxLength={300}
              placeholder="Aquisição de notebooks"
              disabled={create.isPending}
              onChange={(event) => setTitle(event.target.value)}
            />
          </label>
          <button type="submit" disabled={create.isPending}>
            {create.isPending ? "Criando…" : "Criar ETP"}
          </button>
        </form>

        {error && (
          <p className={styles.error} role="alert">
            {error}
          </p>
        )}

        <section
          aria-labelledby="documents-title"
          className={styles.listRegion}
        >
          <h2 id="documents-title">Documentos existentes</h2>
          {documents.isPending && <p role="status">Carregando documentos…</p>}
          {documents.isError && (
            <p role="alert">
              Não foi possível carregar os documentos.{" "}
              <button type="button" onClick={() => void documents.refetch()}>
                Tentar novamente
              </button>
            </p>
          )}
          {documents.isSuccess && documents.data.length === 0 && (
            <p className={styles.empty}>
              Nenhum documento ainda. Crie o primeiro ETP acima.
            </p>
          )}
          {documents.isSuccess && documents.data.length > 0 && (
            <ul className={styles.list}>
              {documents.data.map((summary) => (
                <li key={summary.id}>
                  <Link to={`/documentos/${summary.id}`}>
                    <strong>{summary.title}</strong>
                    <span>
                      Revisão {summary.revision} ·{" "}
                      {formatUpdatedAt(summary.updatedAt)}
                    </span>
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </section>
      </main>
    </div>
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
