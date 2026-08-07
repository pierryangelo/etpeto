import type { SerializedEditorState } from "lexical";
import { useEffect, useMemo, useState } from "react";
import { useParams } from "react-router";

import type { DocumentContent, DocumentSection } from "../../shared/api/client";
import { AppHeader } from "../../shared/ui/AppHeader";
import { RichTextElement } from "./RichTextElement";
import { DocumentPreview } from "./DocumentPreview";
import { useDocumentEditor, type SaveState } from "./useDocumentEditor";
import styles from "./EditorWorkspace.module.css";

const layoutStorageKey = "etpeto:editor-layout:v1";

type MobileMode = "outline" | "editor" | "preview";

type LayoutPreference = {
  outlineOpen: boolean;
  previewOpen: boolean;
};

type EditorWorkspaceProps = {
  documentId?: string;
  compact?: boolean;
};

const defaultPreference: LayoutPreference = {
  outlineOpen: true,
  previewOpen: true,
};

function readPreference(): LayoutPreference {
  try {
    const saved = localStorage.getItem(layoutStorageKey);
    if (!saved) return defaultPreference;
    const candidate: unknown = JSON.parse(saved);
    if (
      typeof candidate === "object" &&
      candidate !== null &&
      "outlineOpen" in candidate &&
      typeof candidate.outlineOpen === "boolean" &&
      "previewOpen" in candidate &&
      typeof candidate.previewOpen === "boolean"
    ) {
      return candidate as LayoutPreference;
    }
  } catch {
    // A corrupt local preference must never block document editing.
  }
  return defaultPreference;
}

function useSmallViewport(): boolean {
  const query = "(max-width: 760px)";
  const [matches, setMatches] = useState(() =>
    typeof window.matchMedia === "function"
      ? window.matchMedia(query).matches
      : false,
  );

  useEffect(() => {
    if (typeof window.matchMedia !== "function") return;
    const media = window.matchMedia(query);
    const update = () => setMatches(media.matches);
    media.addEventListener("change", update);
    return () => media.removeEventListener("change", update);
  }, []);

  return matches;
}

export function EditorWorkspace({ documentId, compact }: EditorWorkspaceProps) {
  const params = useParams();
  const resolvedId = documentId ?? params.id ?? "";
  const smallViewport = useSmallViewport();
  const isCompact = compact ?? smallViewport;
  const [preference, setPreference] = useState(readPreference);
  const [mobileMode, setMobileMode] = useState<MobileMode>("editor");
  const [activeSectionId, setActiveSectionId] = useState("");

  const editor = useDocumentEditor(resolvedId);
  const draft = editor.draft;

  const activeSection = useMemo(() => {
    if (!draft) return undefined;
    return (
      draft.sections.find((section) => section.id === activeSectionId) ??
      draft.sections[0]
    );
  }, [activeSectionId, draft]);

  function updatePreference(next: LayoutPreference) {
    setPreference(next);
    try {
      localStorage.setItem(layoutStorageKey, JSON.stringify(next));
    } catch {
      // Storage is an enhancement; the editor remains usable without it.
    }
  }

  if (editor.isLoading) {
    return (
      <div className={styles.shell}>
        <AppHeader />
        <p className={styles.stateMessage} role="status">
          Carregando documento…
        </p>
      </div>
    );
  }

  if (editor.loadError || !draft || !activeSection) {
    return (
      <div className={styles.shell}>
        <AppHeader />
        <p className={styles.stateMessage} role="alert">
          Não foi possível abrir este documento. Ele pode ter sido removido ou
          pertencer a outra organização.
        </p>
      </div>
    );
  }

  const outline = (
    <DocumentOutline
      sections={draft.sections}
      activeSectionId={activeSection.id}
      compact={isCompact}
      onSelect={(id) => {
        setActiveSectionId(id);
        if (isCompact) setMobileMode("editor");
      }}
      onCollapse={
        isCompact
          ? undefined
          : () => updatePreference({ ...preference, outlineOpen: false })
      }
    />
  );

  const editorPanel = (
    <DocumentEditor
      documentId={resolvedId}
      draft={draft}
      section={activeSection}
      onChangeElement={editor.updateElement}
    />
  );

  const preview = (
    <DocumentPreview
      documentId={resolvedId}
      documentTitle={draft.title}
      revision={editor.revision}
      compact={isCompact}
      onBeforeRender={editor.flush}
      onCollapse={
        isCompact
          ? undefined
          : () => updatePreference({ ...preference, previewOpen: false })
      }
    />
  );

  const identity = (
    <DocumentIdentity
      title={draft.title}
      revision={editor.revision}
      saveState={editor.saveState}
      compact={isCompact}
      onReloadRemote={() => void editor.reloadRemote()}
      onKeepLocal={() => void editor.keepLocal()}
    />
  );

  if (isCompact) {
    return (
      <div className={styles.compactShell}>
        <AppHeader>{identity}</AppHeader>
        <div className={styles.compactContent}>
          {mobileMode === "outline" && outline}
          {mobileMode === "editor" && editorPanel}
          {mobileMode === "preview" && preview}
        </div>
        <MobileNavigation mode={mobileMode} onChange={setMobileMode} />
      </div>
    );
  }

  return (
    <div className={styles.shell}>
      <AppHeader>{identity}</AppHeader>
      <div
        className={styles.workspace}
        data-outline-open={preference.outlineOpen}
        data-preview-open={preference.previewOpen}
      >
        {preference.outlineOpen ? (
          outline
        ) : (
          <CollapsedRail
            label="Roteiro"
            actionLabel="Expandir roteiro"
            side="left"
            onExpand={() =>
              updatePreference({ ...preference, outlineOpen: true })
            }
          />
        )}

        {editorPanel}

        {preference.previewOpen ? (
          preview
        ) : (
          <CollapsedRail
            label="Prévia"
            actionLabel="Expandir prévia oficial"
            side="right"
            onExpand={() =>
              updatePreference({ ...preference, previewOpen: true })
            }
          />
        )}
      </div>
    </div>
  );
}

const saveStateLabels: Record<SaveState, string> = {
  idle: "Sem alterações",
  dirty: "Alterações não salvas",
  saving: "Salvando…",
  saved: "Salvo",
  conflict: "Conflito detectado",
  failed: "Falha ao salvar",
};

function DocumentIdentity({
  title,
  revision,
  saveState,
  compact,
  onReloadRemote,
  onKeepLocal,
}: {
  title: string;
  revision: number;
  saveState: SaveState;
  compact: boolean;
  onReloadRemote: () => void;
  onKeepLocal: () => void;
}) {
  return (
    <div className={styles.identity}>
      <div className={styles.documentIdentity}>
        <strong>{title}</strong>
        {!compact && <span>Revisão {revision}</span>}
      </div>
      <div className={styles.saveState} role="status" data-state={saveState}>
        <span className={styles.saveDot} aria-hidden="true" />
        {saveStateLabels[saveState]}
      </div>
      {saveState === "conflict" && (
        <div
          className={styles.conflictActions}
          role="group"
          aria-label="Resolver conflito"
        >
          <button type="button" onClick={onReloadRemote}>
            Recarregar do servidor
          </button>
          <button type="button" onClick={onKeepLocal}>
            Manter meu texto
          </button>
        </div>
      )}
    </div>
  );
}

function DocumentOutline({
  sections,
  activeSectionId,
  compact,
  onSelect,
  onCollapse,
}: {
  sections: DocumentSection[];
  activeSectionId: string;
  compact: boolean;
  onSelect: (id: string) => void;
  onCollapse?: () => void;
}) {
  const filled = sections.filter(sectionHasContent).length;
  const progress = sections.length
    ? Math.round((filled / sections.length) * 100)
    : 0;

  return (
    <nav
      className={compact ? styles.compactOutline : styles.outline}
      aria-label="Roteiro do documento"
    >
      <div className={styles.panelHeading}>
        <div>
          <span className={styles.panelEyebrow}>Roteiro do ETP</span>
          <strong>{sections.length} seções</strong>
        </div>
        {onCollapse && (
          <button
            className={styles.iconButton}
            type="button"
            aria-label="Recolher roteiro"
            onClick={onCollapse}
          >
            ‹
          </button>
        )}
      </div>
      <ol className={styles.sectionList}>
        {sections.map((section, index) => {
          const active = section.id === activeSectionId;
          const state = active
            ? "active"
            : sectionHasContent(section)
              ? "done"
              : "todo";
          return (
            <li key={section.id} className={styles.sectionItem}>
              <button
                className={styles.sectionButton}
                data-state={state}
                type="button"
                aria-current={active ? "step" : undefined}
                onClick={() => onSelect(section.id)}
              >
                <span className={styles.sectionMarker} aria-hidden="true">
                  {state === "done" ? "✓" : index + 1}
                </span>
                <span>{section.title}</span>
                {active && (
                  <span className={styles.activeLabel}>Em edição</span>
                )}
              </button>
            </li>
          );
        })}
      </ol>
      <div className={styles.outlineProgress}>
        <span>Progresso do roteiro</span>
        <strong>{progress}%</strong>
        <div className={styles.progressTrack} aria-hidden="true">
          <span style={{ width: `${progress}%` }} />
        </div>
      </div>
    </nav>
  );
}

function sectionHasContent(section: DocumentSection): boolean {
  return section.elements.some((element) => {
    const state = element.richText as
      { root?: { children?: unknown[] } } | undefined;
    return Boolean(state?.root?.children?.length);
  });
}

function DocumentEditor({
  documentId,
  draft,
  section,
  onChangeElement,
}: {
  documentId: string;
  draft: DocumentContent;
  section: DocumentSection;
  onChangeElement: (
    sectionId: string,
    elementId: string,
    state: SerializedEditorState,
  ) => void;
}) {
  const position =
    draft.sections.findIndex((item) => item.id === section.id) + 1;

  return (
    <main className={styles.editor}>
      <article className={styles.editorCard}>
        <div className={styles.sectionContext}>
          <span>
            Seção {position} de {draft.sections.length}
          </span>
        </div>
        <h1>{section.title}</h1>
        {section.guidance && (
          <p className={styles.guidance}>{section.guidance}</p>
        )}

        {section.elements.map((element) => (
          <RichTextElement
            key={element.id}
            documentId={documentId}
            elementId={element.id}
            initialState={element.richText as SerializedEditorState | undefined}
            onChange={(state) => onChangeElement(section.id, element.id, state)}
          />
        ))}
      </article>
    </main>
  );
}

function CollapsedRail({
  label,
  actionLabel,
  side,
  onExpand,
}: {
  label: string;
  actionLabel: string;
  side: "left" | "right";
  onExpand: () => void;
}) {
  return (
    <div className={styles.collapsedRail} data-side={side}>
      <button type="button" aria-label={actionLabel} onClick={onExpand}>
        <span aria-hidden="true">{side === "left" ? "›" : "‹"}</span>
        <strong>{label}</strong>
      </button>
    </div>
  );
}

function MobileNavigation({
  mode,
  onChange,
}: {
  mode: MobileMode;
  onChange: (mode: MobileMode) => void;
}) {
  const tabs: Array<{ mode: MobileMode; label: string; icon: string }> = [
    { mode: "outline", label: "Roteiro", icon: "☷" },
    { mode: "editor", label: "Editar", icon: "✎" },
    { mode: "preview", label: "Prévia", icon: "▤" },
  ];

  return (
    <div
      className={styles.mobileNavigation}
      role="tablist"
      aria-label="Área do documento"
    >
      {tabs.map((tab) => (
        <button
          key={tab.mode}
          type="button"
          role="tab"
          aria-selected={mode === tab.mode}
          onClick={() => onChange(tab.mode)}
        >
          <span aria-hidden="true">{tab.icon}</span>
          {tab.label}
        </button>
      ))}
    </div>
  );
}
