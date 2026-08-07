import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router";

import { api } from "../../shared/api/client";
import { AppHeader } from "../../shared/ui/AppHeader";
import { Badge, PageHeading } from "../../shared/ui/controls";
import { DataTable } from "../../shared/ui/DataTable";
import { PageShell } from "../../shared/ui/PageShell";

export function TemplatesPage() {
  const templates = useQuery({
    queryKey: ["templates"],
    queryFn: () => api.listTemplates(),
  });

  return (
    <PageShell header={<AppHeader />}>
      <PageHeading
        eyebrow="Governança do documento"
        title="Roteiros do órgão"
        description="Definem as seções, a orientação e a obrigatoriedade de cada tipo de documento. Um ETP fica preso à revisão com que foi criado."
      />

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
        <DataTable
          columns={["Roteiro", "Em uso", "Documentos", "Rascunho", ""]}
          rows={templates.data.map((template) => ({
            key: template.id,
            cells: [
              { content: template.name, emphasis: true },
              {
                content:
                  template.publishedRevision > 0 ? (
                    <Badge tone="live">
                      Revisão {template.publishedRevision}
                    </Badge>
                  ) : (
                    <Badge>Sem publicação</Badge>
                  ),
              },
              { content: template.documentCount, numeric: true },
              {
                content: template.hasUnpublishedChanges ? (
                  <Badge tone="draft">Alterações não publicadas</Badge>
                ) : (
                  "Em dia"
                ),
              },
              {
                align: "right",
                content: <Link to={`/roteiros/${template.id}`}>Editar</Link>,
              },
            ],
          }))}
        />
      )}
    </PageShell>
  );
}
