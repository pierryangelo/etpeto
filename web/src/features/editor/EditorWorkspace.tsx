import type { SerializedEditorState } from "lexical";
import { useEffect, useMemo, useState } from "react";
import { useParams } from "react-router";

import IconChevronLeft from "~icons/lucide/chevron-left";
import IconChevronRight from "~icons/lucide/chevron-right";

import type { DocumentContent, DocumentSection } from "../../shared/api/client";
import { AppHeader } from "../../shared/ui/AppHeader";
import { RichTextElement } from "./RichTextElement";
import { DocumentPreview } from "./DocumentPreview";
import { PanelDivider } from "./PanelDivider";
import {
  clampOutline,
  clampPreview,
  outlineBounds,
  previewBounds,
  readLayout,
  writeLayout,
  type PanelLayout,
} from "./panelLayout";
import { documentProgress, sectionStatus } from "./sectionProgress";
import { useDocumentEditor, type SaveState } from "./useDocumentEditor";
import styles from "./EditorWorkspace.module.css";

type MobileMode = "outline" | "editor" | "preview";

type EditorWorkspaceProps = {
  documentId?: string;
  compact?: boolean;
};

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
  const [layout, setLayout] = useState(() => readLayout(localStorage));
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

  function updateLayout(change: Partial<PanelLayout>) {
    setLayout((current) => {
      const next = { ...current, ...change };
      writeLayout(localStorage, next);
      return next;
    });
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
      content={draft}
      activeSectionId={activeSection.id}
      compact={isCompact}
      onSelect={(id) => {
        setActiveSectionId(id);
        if (isCompact) setMobileMode("editor");
      }}
      onCollapse={
        isCompact ? undefined : () => updateLayout({ outlineOpen: false })
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
        isCompact ? undefined : () => updateLayout({ previewOpen: false })
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
        style={
          {
            "--outline-width": `${layout.outlineOpen ? layout.outlineWidth : 0}px`,
            "--preview-width": `${layout.previewOpen ? layout.previewWidth : 0}px`,
          } as React.CSSProperties
        }
        data-outline-open={layout.outlineOpen}
        data-preview-open={layout.previewOpen}
      >
        {layout.outlineOpen ? (
          <>
            {outline}
            <PanelDivider
              label="Redimensionar roteiro"
              width={layout.outlineWidth}
              min={outlineBounds.min}
              max={outlineBounds.max}
              grow="right"
              onResize={(width) =>
                updateLayout({ outlineWidth: clampOutline(width) })
              }
              onReset={() =>
                updateLayout({ outlineWidth: outlineBounds.default })
              }
            />
          </>
        ) : (
          <CollapsedRail
            label="Roteiro"
            actionLabel="Expandir roteiro"
            side="left"
            onExpand={() => updateLayout({ outlineOpen: true })}
          />
        )}

        {editorPanel}

        {layout.previewOpen ? (
          <>
            <PanelDivider
              label="Redimensionar prévia oficial"
              width={layout.previewWidth}
              min={previewBounds.min}
              max={previewBounds.max}
              grow="left"
              onResize={(width) =>
                updateLayout({ previewWidth: clampPreview(width) })
              }
              onReset={() =>
                updateLayout({ previewWidth: previewBounds.default })
              }
            />
            {preview}
          </>
        ) : (
          <CollapsedRail
            label="Prévia"
            actionLabel="Expandir prévia oficial"
            side="right"
            onExpand={() => updateLayout({ previewOpen: true })}
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
  content,
  activeSectionId,
  compact,
  onSelect,
  onCollapse,
}: {
  content: DocumentContent;
  activeSectionId: string;
  compact: boolean;
  onSelect: (id: string) => void;
  onCollapse?: () => void;
}) {
  const sections = content.sections;
  const { statuses, percent, missingRequired } = documentProgress(content);

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
            <IconChevronLeft />
          </button>
        )}
      </div>
      <ol className={styles.sectionList}>
        {statuses.map((status, index) => {
          const active = status.section.id === activeSectionId;
          const state = active ? "active" : status.filled ? "done" : "todo";
          return (
            <li key={status.section.id} className={styles.sectionItem}>
              <button
                className={styles.sectionButton}
                data-state={state}
                type="button"
                aria-current={active ? "step" : undefined}
                onClick={() => onSelect(status.section.id)}
              >
                <span className={styles.sectionMarker} aria-hidden="true">
                  {status.filled ? "✓" : index + 1}
                </span>
                <span className={styles.sectionName}>
                  {status.section.title}
                </span>
                <span
                  className={styles.sectionRequirement}
                  data-pending={
                    status.required && !status.filled ? "" : undefined
                  }
                >
                  {status.required ? "Obrig." : "Opcional"}
                </span>
              </button>
            </li>
          );
        })}
      </ol>
      <div className={styles.outlineProgress}>
        <span>Preenchimento</span>
        <strong>{percent}%</strong>
        <div className={styles.progressTrack} aria-hidden="true">
          <span style={{ width: `${percent}%` }} />
        </div>
        {/* Naming the pending section beats a percentage: it says what to do
            next instead of how far along the document is. */}
        {missingRequired.length > 0 && (
          <p className={styles.pending}>
            {missingRequired.length === 1
              ? `1 seção obrigatória ainda vazia: ${missingRequired[0]}.`
              : `${missingRequired.length} seções obrigatórias ainda vazias: ${missingRequired.join(", ")}.`}
          </p>
        )}
      </div>
    </nav>
  );
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
  const status = sectionStatus(draft, section);

  return (
    <main className={styles.editor}>
      <div className={styles.editorContext}>
        <span>
          Seção {status.position} de {draft.sections.length}
        </span>
        <span className={styles.contextDot} aria-hidden="true" />
        <span data-required={status.required ? "" : undefined}>
          {status.required ? "Obrigatória" : "Opcional"}
        </span>
        <span className={styles.contextDot} aria-hidden="true" />
        <span>
          {status.words} {status.words === 1 ? "palavra" : "palavras"}
        </span>
      </div>
      <div className={styles.editorScroll}>
        <article className={styles.editorCard}>
          <h1>{section.title}</h1>
          {section.guidance && (
            <p className={styles.guidance}>{section.guidance}</p>
          )}

          {section.elements.map((element) => (
            <RichTextElement
              key={element.id}
              documentId={documentId}
              elementId={element.id}
              initialState={
                element.richText as SerializedEditorState | undefined
              }
              onChange={(state) =>
                onChangeElement(section.id, element.id, state)
              }
            />
          ))}
        </article>
      </div>
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
        {side === "left" ? <IconChevronRight /> : <IconChevronLeft />}
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
