import { useMemo, useState } from "react";
import { useParams } from "react-router";

import type { BlueprintSection } from "../../shared/api/client";
import { AppHeader } from "../../shared/ui/AppHeader";
import { SectionTree, type TreeNode } from "../../shared/ui/SectionTree";
import {
  Badge,
  Button,
  Chip,
  DangerZone,
  DetailPane,
  Divider,
  NoticeBar,
  Panel,
  PanelActions,
  Switch,
  TextAreaField,
  TextField,
} from "../../shared/ui/controls";
import { WorkspaceShell } from "../../shared/ui/PageShell";
import {
  addSection,
  canIndent,
  canOutdent,
  findSection,
  indentSection,
  moveSection,
  newSection,
  outdentSection,
  removeSection,
  updateSection,
} from "./blueprintTree";
import { useTemplateDraft, type DraftState } from "./useTemplateDraft";

const draftLabels: Record<DraftState, string> = {
  idle: "Sem alterações",
  dirty: "Alterações não salvas",
  saving: "Salvando…",
  saved: "Rascunho salvo",
  conflict: "Conflito detectado",
  failed: "Falha ao salvar",
};

const elementLabels: Record<string, string> = {
  richText: "Texto rico",
  priceResearch: "Pesquisa de preços",
  riskMap: "Mapa de riscos",
  table: "Tabela",
};

export function TemplateEditorPage() {
  const params = useParams();
  const templateId = params.id ?? "";
  const draft = useTemplateDraft(templateId);
  const [selectedKey, setSelectedKey] = useState("");

  const flattened = useMemo(
    () => (draft.blueprint ? flatten(draft.blueprint.sections) : []),
    [draft.blueprint],
  );
  const nodes = useMemo(
    () => (draft.blueprint ? toNodes(draft.blueprint.sections) : []),
    [draft.blueprint],
  );
  const selected =
    (draft.blueprint && findSection(draft.blueprint.sections, selectedKey)) ??
    flattened[0]?.section;

  if (draft.isLoading) {
    return (
      <Shell>
        <p className="p-6" role="status">
          Carregando roteiro…
        </p>
      </Shell>
    );
  }
  if (draft.loadError || !draft.blueprint || !selected) {
    return (
      <Shell>
        <p className="p-6" role="alert">
          Não foi possível abrir este roteiro.
        </p>
      </Shell>
    );
  }

  const sections = draft.blueprint.sections;
  const publishedRevision = draft.template?.publishedRevision ?? 0;

  return (
    <Shell>
      {/* The draft bar states, at all times, what the authors are still using. */}
      <NoticeBar
        actions={
          <>
            <Button variant="quiet" small onClick={() => void draft.discard()}>
              Descartar alterações
            </Button>
            <Button
              variant="primary"
              small
              disabled={draft.publish.isPending}
              onClick={() => draft.publish.mutate()}
            >
              {draft.publish.isPending
                ? "Publicando…"
                : `Publicar revisão ${publishedRevision + 1}`}
            </Button>
          </>
        }
      >
        <strong>Rascunho</strong>
        <span aria-hidden="true">·</span>
        <span role="status">{draftLabels[draft.state]}</span>
        <span className="ds-meta">
          {publishedRevision > 0
            ? `os autores continuam na revisão ${publishedRevision}`
            : "nenhuma revisão publicada ainda"}
        </span>
      </NoticeBar>

      {draft.publish.isError && (
        <NoticeBar tone="danger">
          <span role="alert">
            Não foi possível publicar. Se o rascunho estiver igual à revisão em
            uso, altere algo antes de publicar.
          </span>
        </NoticeBar>
      )}

      <div className="flex min-h-0 flex-1">
        <Panel eyebrow="Estrutura" title={`${flattened.length} seções`}>
          <SectionTree
            nodes={nodes}
            selectedId={selected.key}
            label="Estrutura do roteiro"
            onSelect={setSelectedKey}
          />
          <PanelActions>
            <Button
              small
              onClick={() =>
                draft.apply((c) => addSection(c, null, newSection()))
              }
            >
              ＋ Seção
            </Button>
            <Button
              small
              onClick={() =>
                draft.apply((c) =>
                  addSection(c, selected.key, newSection("Nova subseção")),
                )
              }
            >
              ＋ Subseção
            </Button>
          </PanelActions>
        </Panel>

        <Divider />

        <DetailPane>
          <div className="flex items-center gap-2">
            <span className="ds-eyebrow">Seção selecionada</span>
            <Badge tone={selected.required ? "live" : "muted"}>
              {selected.required ? "Obrigatória" : "Opcional"}
            </Badge>
            <span className="ml-auto flex gap-1.5">
              <Button
                small
                onClick={() =>
                  draft.apply((c) => moveSection(c, selected.key, "up"))
                }
              >
                ↑
              </Button>
              <Button
                small
                onClick={() =>
                  draft.apply((c) => moveSection(c, selected.key, "down"))
                }
              >
                ↓
              </Button>
              <Button
                small
                disabled={!canIndent(sections, selected.key)}
                onClick={() =>
                  draft.apply((c) => indentSection(c, selected.key))
                }
              >
                → Recuar
              </Button>
              <Button
                small
                disabled={!canOutdent(sections, selected.key)}
                onClick={() =>
                  draft.apply((c) => outdentSection(c, selected.key))
                }
              >
                ← Promover
              </Button>
            </span>
          </div>

          <TextField
            label="Título"
            value={selected.title}
            onChange={(title) =>
              draft.apply((c) => updateSection(c, selected.key, { title }))
            }
          />

          <TextAreaField
            label="Orientação para quem escreve"
            help="Aparece abaixo do título da seção no editor, e não vai para o PDF."
            value={selected.guidance ?? ""}
            onChange={(guidance) =>
              draft.apply((c) => updateSection(c, selected.key, { guidance }))
            }
          />

          <Switch
            checked={selected.required}
            title="Seção obrigatória"
            description="Impede finalizar o ETP enquanto estiver vazia. Não obriga as subseções."
            onChange={(required) =>
              draft.apply((c) => updateSection(c, selected.key, { required }))
            }
          />
          <Switch
            checked={selected.allowRename}
            title="Permitir que o autor renomeie"
            description="O significado da seção é preservado mesmo com outro título."
            onChange={(allowRename) =>
              draft.apply((c) =>
                updateSection(c, selected.key, { allowRename }),
              )
            }
          />

          <div className="grid gap-1.5">
            <span className="ds-label">Conteúdo permitido</span>
            <div className="flex flex-wrap gap-1.5">
              {Object.entries(elementLabels).map(([type, label]) => {
                const enabled = selected.allowedElements.includes(type);
                return (
                  <Chip
                    key={type}
                    pressed={enabled}
                    onToggle={() =>
                      draft.apply((c) =>
                        updateSection(c, selected.key, {
                          allowedElements: enabled
                            ? selected.allowedElements.filter(
                                (item) => item !== type,
                              )
                            : [...selected.allowedElements, type],
                        }),
                      )
                    }
                  >
                    {label}
                  </Chip>
                );
              })}
            </div>
            <span className="ds-help">
              Só o que estiver marcado pode ser inserido pelo autor nesta seção.
            </span>
          </div>

          <DangerZone
            note="Remover não afeta documentos já criados: eles seguem na revisão com que nasceram."
            action={
              <Button
                variant="danger"
                small
                onClick={() => {
                  draft.apply((c) => removeSection(c, selected.key));
                  setSelectedKey("");
                }}
              >
                Remover seção
              </Button>
            }
          />
        </DetailPane>
      </div>
    </Shell>
  );
}

function Shell({ children }: { children: React.ReactNode }) {
  return <WorkspaceShell header={<AppHeader />}>{children}</WorkspaceShell>;
}

function toNodes(sections: BlueprintSection[]): TreeNode[] {
  return sections.map((section) => ({
    id: section.key,
    title: section.title,
    badge: { label: section.required ? "Obrig." : "Opcional" },
    children: toNodes(section.sections ?? []),
  }));
}

function flatten(
  sections: BlueprintSection[],
  prefix = "",
  level = 1,
): { section: BlueprintSection; number: string; level: number }[] {
  return sections.flatMap((section, index) => {
    const number = prefix ? `${prefix}.${index + 1}` : String(index + 1);
    return [
      { section, number, level },
      ...flatten(section.sections ?? [], number, level + 1),
    ];
  });
}
