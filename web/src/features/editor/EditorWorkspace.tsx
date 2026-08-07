import { useEffect, useState } from "react";

import styles from "./EditorWorkspace.module.css";

const layoutStorageKey = "etpeto:editor-layout:v1";

const sections = [
  { label: "Necessidade", state: "done" },
  { label: "Requisitos", state: "active" },
  { label: "Pesquisa de preços", state: "todo" },
  { label: "Mapa de riscos", state: "todo" },
  { label: "Conclusão", state: "todo" },
] as const;

type MobileMode = "outline" | "editor" | "preview";

type LayoutPreference = {
  outlineOpen: boolean;
  previewOpen: boolean;
};

type EditorWorkspaceProps = {
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

export function EditorWorkspace({ compact }: EditorWorkspaceProps) {
  const smallViewport = useSmallViewport();
  const isCompact = compact ?? smallViewport;
  const [preference, setPreference] = useState(readPreference);
  const [mobileMode, setMobileMode] = useState<MobileMode>("editor");

  function updatePreference(next: LayoutPreference) {
    setPreference(next);
    try {
      localStorage.setItem(layoutStorageKey, JSON.stringify(next));
    } catch {
      // Storage is an enhancement; the editor remains usable without it.
    }
  }

  if (isCompact) {
    return (
      <div className={styles.compactShell}>
        <DocumentHeader compact />
        <div className={styles.compactContent}>
          {mobileMode === "outline" && <DocumentOutline compact />}
          {mobileMode === "editor" && <DocumentEditor />}
          {mobileMode === "preview" && <DocumentPreview compact />}
        </div>
        <MobileNavigation mode={mobileMode} onChange={setMobileMode} />
      </div>
    );
  }

  return (
    <div className={styles.shell}>
      <DocumentHeader />
      <div
        className={styles.workspace}
        data-outline-open={preference.outlineOpen}
        data-preview-open={preference.previewOpen}
      >
        {preference.outlineOpen ? (
          <DocumentOutline
            onCollapse={() =>
              updatePreference({ ...preference, outlineOpen: false })
            }
          />
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

        <DocumentEditor />

        {preference.previewOpen ? (
          <DocumentPreview
            onCollapse={() =>
              updatePreference({ ...preference, previewOpen: false })
            }
          />
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

function DocumentHeader({ compact = false }: { compact?: boolean }) {
  return (
    <header className={styles.header}>
      <a className={styles.product} href="/" aria-label="Etpeto — documentos">
        <span className={styles.productMark} aria-hidden="true">
          E
        </span>
        {!compact && <span>Etpeto</span>}
      </a>
      <span className={styles.headerDivider} aria-hidden="true" />
      <div className={styles.documentIdentity}>
        <strong>Aquisição de notebooks</strong>
        {!compact && <span>ETP 014/2026</span>}
      </div>
      <div className={styles.saveState} role="status">
        <span className={styles.saveDot} aria-hidden="true" />
        {compact ? "Salvo" : "Salvo agora"}
      </div>
      {!compact && (
        <button className={styles.secondaryButton} type="button" disabled>
          Finalizar
        </button>
      )}
      <button className={styles.primaryButton} type="button" disabled>
        {compact ? "PDF" : "Exportar PDF"}
      </button>
    </header>
  );
}

function DocumentOutline({
  compact = false,
  onCollapse,
}: {
  compact?: boolean;
  onCollapse?: () => void;
}) {
  return (
    <nav
      className={compact ? styles.compactOutline : styles.outline}
      aria-label="Roteiro do documento"
    >
      <div className={styles.panelHeading}>
        <div>
          <span className={styles.panelEyebrow}>Roteiro do ETP</span>
          <strong>5 seções</strong>
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
        {sections.map((section, index) => (
          <li key={section.label} className={styles.sectionItem}>
            <button
              className={styles.sectionButton}
              data-state={section.state}
              type="button"
              aria-current={section.state === "active" ? "step" : undefined}
            >
              <span className={styles.sectionMarker} aria-hidden="true">
                {section.state === "done" ? "✓" : index + 1}
              </span>
              <span>{section.label}</span>
              {section.state === "active" && (
                <span className={styles.activeLabel}>Em edição</span>
              )}
            </button>
          </li>
        ))}
      </ol>
      <div className={styles.outlineProgress}>
        <span>Progresso do roteiro</span>
        <strong>20%</strong>
        <div className={styles.progressTrack} aria-hidden="true">
          <span />
        </div>
      </div>
    </nav>
  );
}

function DocumentEditor() {
  return (
    <main className={styles.editor}>
      <article className={styles.editorCard}>
        <div className={styles.sectionContext}>
          <span>Seção 2 de 5</span>
          <span className={styles.requiredLabel}>Obrigatória</span>
        </div>
        <h1>Requisitos da contratação</h1>
        <p className={styles.guidance}>
          Descreva as condições indispensáveis para atender à necessidade,
          evitando direcionamento de marca ou solução.
        </p>

        <div className={styles.toolbar} role="toolbar" aria-label="Formatação">
          <button type="button" aria-label="Negrito">
            <strong>B</strong>
          </button>
          <button type="button" aria-label="Itálico">
            <em>I</em>
          </button>
          <button type="button" aria-label="Lista com marcadores">
            ☷
          </button>
          <span aria-hidden="true" />
          <button type="button" aria-label="Inserir link">
            ↗
          </button>
        </div>

        <div className={styles.textSurface} aria-label="Conteúdo da seção">
          <p>
            Os equipamentos deverão atender às atividades administrativas e
            técnicas, com desempenho compatível com o ciclo de uso previsto.
          </p>
          <p>
            A contratação deverá incluir garantia, suporte e critérios de
            eficiência energética verificáveis.
          </p>
        </div>

        <section className={styles.structuredBlock}>
          <div className={styles.blockIcon} aria-hidden="true">
            ≋
          </div>
          <div>
            <span className={styles.blockType}>Elemento estruturado</span>
            <strong>Pesquisa de preços</strong>
            <p>3 fontes cadastradas · estimativa de R$ 428.000,00</p>
          </div>
          <button type="button">Abrir</button>
        </section>

        <button className={styles.addElementButton} type="button">
          <span aria-hidden="true">＋</span>
          Adicionar elemento
        </button>
      </article>
    </main>
  );
}

function DocumentPreview({
  compact = false,
  onCollapse,
}: {
  compact?: boolean;
  onCollapse?: () => void;
}) {
  return (
    <aside
      className={compact ? styles.compactPreview : styles.preview}
      aria-label="Prévia oficial"
    >
      <div className={styles.panelHeading}>
        <div>
          <span className={styles.panelEyebrow}>Documento gerado</span>
          <strong>Prévia oficial</strong>
        </div>
        {onCollapse && (
          <button
            className={styles.iconButton}
            type="button"
            aria-label="Recolher prévia oficial"
            onClick={onCollapse}
          >
            ›
          </button>
        )}
      </div>
      <div className={styles.previewMeta}>
        <span>PDF · Typst</span>
        <button type="button" disabled>
          Atualizar
        </button>
      </div>
      <div className={styles.paper} aria-hidden="true">
        <div className={styles.paperSeal}>Etpeto</div>
        <p>ESTUDO TÉCNICO PRELIMINAR</p>
        <h2>Aquisição de equipamentos de tecnologia</h2>
        <div className={styles.paperRule} />
        <h3>2. Requisitos da contratação</h3>
        <span className={styles.paperLine} />
        <span className={styles.paperLine} />
        <span className={`${styles.paperLine} ${styles.paperLineShort}`} />
        <span className={styles.paperLine} />
        <div className={styles.paperTable}>
          <span />
          <span />
          <span />
        </div>
        <small>2</small>
      </div>
      <p className={styles.previewNotice}>
        O PDF será atualizado depois do próximo salvamento.
      </p>
    </aside>
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
