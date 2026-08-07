import { useMemo, useState } from "react";
import { useParams } from "react-router";

import type { BlueprintSection } from "../../shared/api/client";
import { AppHeader } from "../../shared/ui/AppHeader";
import { Pill } from "./TemplatesPage";
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
      <div className="flex items-center gap-2.5 border-b border-line bg-accent-soft px-5 py-2 text-12px text-accent-strong">
        <strong>Rascunho</strong>
        <span aria-hidden="true">·</span>
        <span role="status">{draftLabels[draft.state]}</span>
        <span className="text-subtle">
          {publishedRevision > 0
            ? `os autores continuam na revisão ${publishedRevision}`
            : "nenhuma revisão publicada ainda"}
        </span>
        <span className="ml-auto flex items-center gap-1.5">
          <button
            type="button"
            className="rounded-1 border border-transparent px-2 py-1 text-12px text-text hover:border-line"
            onClick={() => void draft.discard()}
          >
            Descartar alterações
          </button>
          <button
            type="button"
            className="rounded-1 border border-accent-strong bg-accent px-2.5 py-1 text-12px font-600 text-white disabled:opacity-60"
            disabled={draft.publish.isPending}
            onClick={() => draft.publish.mutate()}
          >
            {draft.publish.isPending
              ? "Publicando…"
              : `Publicar revisão ${publishedRevision + 1}`}
          </button>
        </span>
      </div>

      {draft.publish.isError && (
        <p
          className="border-b border-line px-5 py-2 text-12px text-danger"
          role="alert"
        >
          Não foi possível publicar. Se o rascunho estiver igual à revisão em
          uso, altere algo antes de publicar.
        </p>
      )}

      <div className="flex min-h-0 flex-1">
        <nav
          className="flex w-75 flex-none flex-col overflow-auto bg-panel"
          aria-label="Estrutura do roteiro"
        >
          <div className="border-b border-line px-3 py-2.5">
            <span className="text-10px font-700 tracking-wider uppercase text-subtle">
              Estrutura
            </span>
            <strong className="mt-0.5 block text-12.5px">
              {flattened.length} seções
            </strong>
          </div>

          <ul className="m-0 flex list-none flex-col gap-px p-1">
            {flattened.map((entry) => (
              <li key={entry.section.key}>
                <button
                  type="button"
                  aria-current={
                    entry.section.key === selected.key ? "true" : undefined
                  }
                  onClick={() => setSelectedKey(entry.section.key)}
                  className="grid w-full grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-2 rounded-1 px-2 py-1.5 text-left text-12.5px text-text hover:bg-paper aria-[current]:bg-paper aria-[current]:font-600 aria-[current]:text-ink aria-[current]:shadow-[inset_2px_0_0_var(--color-accent)]"
                  style={{ paddingLeft: `${(entry.level - 1) * 14 + 8}px` }}
                >
                  <span className="text-10px font-700 tabular-nums text-subtle">
                    {entry.number}
                  </span>
                  <span className="truncate">{entry.section.title}</span>
                  <span
                    className="text-9px font-700 tracking-wide uppercase text-subtle data-[on]:text-accent-strong"
                    data-on={entry.section.required ? "" : undefined}
                  >
                    {entry.section.required ? "Obrig." : "Opcional"}
                  </span>
                </button>
              </li>
            ))}
          </ul>

          <div className="mt-auto flex gap-1.5 border-t border-line p-2">
            <SmallButton
              onClick={() =>
                draft.apply((current) =>
                  addSection(current, null, newSection()),
                )
              }
            >
              ＋ Seção
            </SmallButton>
            <SmallButton
              onClick={() =>
                draft.apply((current) =>
                  addSection(
                    current,
                    selected.key,
                    newSection("Nova subseção"),
                  ),
                )
              }
            >
              ＋ Subseção
            </SmallButton>
          </div>
        </nav>

        <div className="w-px flex-none bg-line" aria-hidden="true" />

        <div className="flex min-w-0 flex-1 flex-col gap-3.5 overflow-auto bg-paper p-5">
          <div className="flex items-center gap-2">
            <span className="text-10px font-700 tracking-wider uppercase text-subtle">
              Seção selecionada
            </span>
            <Pill tone={selected.required ? "live" : "old"}>
              {selected.required ? "Obrigatória" : "Opcional"}
            </Pill>
            <span className="ml-auto flex gap-1.5">
              <SmallButton
                onClick={() =>
                  draft.apply((c) => moveSection(c, selected.key, "up"))
                }
              >
                ↑
              </SmallButton>
              <SmallButton
                onClick={() =>
                  draft.apply((c) => moveSection(c, selected.key, "down"))
                }
              >
                ↓
              </SmallButton>
              <SmallButton
                disabled={!canIndent(sections, selected.key)}
                onClick={() =>
                  draft.apply((c) => indentSection(c, selected.key))
                }
              >
                → Recuar
              </SmallButton>
              <SmallButton
                disabled={!canOutdent(sections, selected.key)}
                onClick={() =>
                  draft.apply((c) => outdentSection(c, selected.key))
                }
              >
                ← Promover
              </SmallButton>
            </span>
          </div>

          <Field label="Título">
            <input
              className="w-full rounded-1 border border-line px-2 py-1.5 text-13px text-ink"
              value={selected.title}
              onChange={(event) =>
                draft.apply((c) =>
                  updateSection(c, selected.key, { title: event.target.value }),
                )
              }
            />
          </Field>

          <Field
            label="Orientação para quem escreve"
            help="Aparece abaixo do título da seção no editor, e não vai para o PDF."
          >
            <textarea
              className="min-h-16 w-full resize-y rounded-1 border border-line px-2 py-1.5 text-13px leading-relaxed text-ink"
              value={selected.guidance ?? ""}
              onChange={(event) =>
                draft.apply((c) =>
                  updateSection(c, selected.key, {
                    guidance: event.target.value,
                  }),
                )
              }
            />
          </Field>

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

          <Field
            label="Conteúdo permitido"
            help="Só o que estiver marcado pode ser inserido pelo autor nesta seção."
          >
            <div className="flex flex-wrap gap-1.5">
              {Object.entries(elementLabels).map(([type, label]) => {
                const enabled = selected.allowedElements.includes(type);
                return (
                  <button
                    key={type}
                    type="button"
                    aria-pressed={enabled}
                    onClick={() =>
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
                    className="rounded-1 border border-line px-2 py-1 text-11.5px text-subtle aria-pressed:border-accent-line aria-pressed:bg-accent-soft aria-pressed:text-accent-strong"
                  >
                    {label}
                  </button>
                );
              })}
            </div>
          </Field>

          <div className="mt-2 border-t border-line pt-3">
            <button
              type="button"
              className="rounded-1 border border-line px-2 py-1 text-12px text-danger hover:border-danger"
              onClick={() => {
                draft.apply((c) => removeSection(c, selected.key));
                setSelectedKey("");
              }}
            >
              Remover seção
            </button>
            <p className="mt-1.5 text-11px text-subtle">
              Remover não afeta documentos já criados: eles seguem na revisão
              com que nasceram.
            </p>
          </div>
        </div>
      </div>
    </Shell>
  );
}

function Shell({ children }: { children: React.ReactNode }) {
  return (
    <div className="grid h-screen grid-rows-[auto_minmax(0,1fr)] overflow-hidden bg-canvas">
      <AppHeader />
      <div className="flex min-h-0 flex-col">{children}</div>
    </div>
  );
}

function Field({
  label,
  help,
  children,
}: {
  label: string;
  help?: string;
  children: React.ReactNode;
}) {
  return (
    <label className="grid gap-1.5">
      <span className="text-10.5px font-700 tracking-wide uppercase text-subtle">
        {label}
      </span>
      {children}
      {help && (
        <span className="text-11px leading-snug text-subtle">{help}</span>
      )}
    </label>
  );
}

function Switch({
  checked,
  title,
  description,
  onChange,
}: {
  checked: boolean;
  title: string;
  description: string;
  onChange: (value: boolean) => void;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={title}
      onClick={() => onChange(!checked)}
      className="flex items-center gap-2.5 border-t border-line pt-2.5 text-left"
    >
      <span
        aria-hidden="true"
        data-on={checked ? "" : undefined}
        className="relative h-4.5 w-7.5 flex-none rounded-full bg-line data-[on]:bg-accent after:absolute after:left-0.5 after:top-0.5 after:size-3.5 after:rounded-full after:bg-white data-[on]:after:left-auto data-[on]:after:right-0.5"
      />
      <span>
        <strong className="block text-12.5px text-ink">{title}</strong>
        <span className="text-11px text-subtle">{description}</span>
      </span>
    </button>
  );
}

function SmallButton({
  children,
  disabled,
  onClick,
}: {
  children: React.ReactNode;
  disabled?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      className="rounded-1 border border-line bg-paper px-2 py-1 text-11.5px text-text hover:border-accent-line hover:text-accent-strong disabled:opacity-45 disabled:hover:border-line disabled:hover:text-text"
    >
      {children}
    </button>
  );
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
