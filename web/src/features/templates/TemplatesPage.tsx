import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router";

import { api } from "../../shared/api/client";
import { AppHeader } from "../../shared/ui/AppHeader";

export function TemplatesPage() {
  const templates = useQuery({
    queryKey: ["templates"],
    queryFn: () => api.listTemplates(),
  });

  return (
    <div className="min-h-screen flex flex-col bg-canvas">
      <AppHeader />
      <main className="mx-auto w-full max-w-4xl flex flex-col gap-5 px-5 py-6">
        <div>
          <span className="text-10px font-700 tracking-wider uppercase text-accent-strong">
            Governança do documento
          </span>
          <h1 className="mt-1 mb-1 text-19px font-680 tracking-tight text-ink">
            Roteiros do órgão
          </h1>
          <p className="m-0 max-w-2xl text-12.5px text-subtle">
            Definem as seções, a orientação e a obrigatoriedade de cada tipo de
            documento. Um ETP fica preso à revisão com que foi criado.
          </p>
        </div>

        {templates.isPending && <p role="status">Carregando roteiros…</p>}
        {templates.isError && (
          <p role="alert">
            Não foi possível carregar os roteiros.{" "}
            <button type="button" onClick={() => void templates.refetch()}>
              Tentar novamente
            </button>
          </p>
        )}

        {templates.isSuccess && (
          <table className="w-full border-collapse overflow-hidden rounded-2 border border-line bg-paper text-12.5px">
            <thead>
              <tr className="bg-panel">
                <Th>Roteiro</Th>
                <Th>Em uso</Th>
                <Th>Documentos</Th>
                <Th>Rascunho</Th>
                <Th />
              </tr>
            </thead>
            <tbody>
              {templates.data.map((template) => (
                <tr
                  key={template.id}
                  className="border-b border-line last:border-b-0"
                >
                  <td className="px-3 py-2.5 font-550 text-ink">
                    {template.name}
                  </td>
                  <td className="px-3 py-2.5">
                    {template.publishedRevision > 0 ? (
                      <Pill tone="live">
                        Revisão {template.publishedRevision}
                      </Pill>
                    ) : (
                      <Pill tone="old">Sem publicação</Pill>
                    )}
                  </td>
                  <td className="px-3 py-2.5 tabular-nums text-text">
                    {template.documentCount}
                  </td>
                  <td className="px-3 py-2.5">
                    {template.hasUnpublishedChanges ? (
                      <Pill tone="draft">Alterações não publicadas</Pill>
                    ) : (
                      <span className="text-subtle">Em dia</span>
                    )}
                  </td>
                  <td className="px-3 py-2.5 text-right">
                    <Link
                      className="rounded-1 border border-line px-2 py-1 text-12px text-text no-underline hover:border-accent-line hover:text-accent-strong"
                      to={`/roteiros/${template.id}`}
                    >
                      Editar
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </main>
    </div>
  );
}

function Th({ children }: { children?: React.ReactNode }) {
  return (
    <th className="border-b border-line px-3 py-2 text-left text-10px font-700 tracking-wider uppercase text-subtle">
      {children}
    </th>
  );
}

export function Pill({
  tone,
  children,
}: {
  tone: "live" | "draft" | "old";
  children: React.ReactNode;
}) {
  const tones = {
    live: "bg-success-soft text-success",
    draft: "bg-accent-soft text-accent-strong",
    old: "bg-panel text-subtle",
  } as const;
  return (
    <span
      className={`inline-block rounded-0.5 px-1.5 py-0.5 text-9.5px font-700 tracking-wide uppercase ${tones[tone]}`}
    >
      {children}
    </span>
  );
}
