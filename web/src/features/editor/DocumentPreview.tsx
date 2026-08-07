import { useEffect, useRef, useState } from "react";

import IconChevronRight from "~icons/lucide/chevron-right";
import IconDownload from "~icons/lucide/download";
import IconRefresh from "~icons/lucide/refresh-cw";

import { api } from "../../shared/api/client";
import styles from "./EditorWorkspace.module.css";

type DocumentPreviewProps = {
  documentId: string;
  documentTitle?: string;
  revision: number;
  compact?: boolean;
  onBeforeRender?: () => void;
  onCollapse?: () => void;
};

type PreviewState = "empty" | "rendering" | "ready" | "failed";

/**
 * Shows the official PDF produced by the same Typst pipeline used for export.
 *
 * The rendered revision is tracked so the panel can say when the PDF is behind
 * the document, and the previous PDF is kept on screen while a new one renders
 * or fails — a renderer outage degrades the preview, never the editing.
 */
export function DocumentPreview({
  documentId,
  documentTitle = "documento",
  revision,
  compact = false,
  onBeforeRender,
  onCollapse,
}: DocumentPreviewProps) {
  const [state, setState] = useState<PreviewState>("empty");
  const [pdfUrl, setPdfUrl] = useState("");
  const [renderedRevision, setRenderedRevision] = useState(0);
  const urlRef = useRef("");

  // Object URLs are a manual resource: without revoking them the browser keeps
  // every rendered PDF alive for the lifetime of the tab.
  useEffect(() => {
    return () => {
      if (urlRef.current) URL.revokeObjectURL(urlRef.current);
    };
  }, []);

  async function render() {
    onBeforeRender?.();
    setState("rendering");
    try {
      const blob = await api.previewDocument(documentId);
      const nextUrl = URL.createObjectURL(blob);
      if (urlRef.current) URL.revokeObjectURL(urlRef.current);
      urlRef.current = nextUrl;
      setPdfUrl(nextUrl);
      setRenderedRevision(revision);
      setState("ready");
    } catch {
      setState("failed");
    }
  }

  const outdated = state === "ready" && renderedRevision !== revision;

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
            <IconChevronRight />
          </button>
        )}
      </div>
      <div className={styles.previewMeta}>
        <span>PDF · Typst</span>
        <div className={styles.previewActions}>
          <button
            type="button"
            onClick={() => void render()}
            disabled={state === "rendering"}
          >
            <IconRefresh
              data-spinning={state === "rendering" ? "" : undefined}
            />
            {state === "rendering" ? "Gerando…" : "Atualizar"}
          </button>
          <button
            type="button"
            className={styles.downloadLink}
            disabled={state !== "ready" || !pdfUrl}
            onClick={() => downloadPdf(pdfUrl, documentTitle)}
          >
            <IconDownload />
            Exportar PDF
          </button>
        </div>
      </div>

      {state === "empty" && (
        <p className={styles.previewNotice}>
          Gere a prévia para ver o PDF oficial desta revisão.
        </p>
      )}
      {state === "failed" && (
        <p className={styles.previewNotice} role="alert">
          A renderização falhou. Seu texto continua salvo — tente atualizar
          novamente.
        </p>
      )}
      {pdfUrl && (
        <>
          <iframe
            className={styles.previewFrame}
            src={pdfUrl}
            title="Prévia oficial em PDF"
          />
          {outdated && (
            <p className={styles.previewNotice} role="status">
              O PDF mostra a revisão {renderedRevision}; o documento está na{" "}
              {revision}.
            </p>
          )}
        </>
      )}
    </aside>
  );
}

/**
 * Downloads the PDF already rendered by the server. Nothing is re-rendered on
 * export, so the exported file is byte-identical to what the preview showed.
 */
function downloadPdf(url: string, title: string) {
  if (!url) return;
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = `${safeFileName(title)}.pdf`;
  document.body.append(anchor);
  anchor.click();
  anchor.remove();
}

function safeFileName(title: string): string {
  const normalized = title
    .normalize("NFD")
    .replace(/[̀-ͯ]/g, "")
    .replace(/[^a-zA-Z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .toLowerCase()
    .slice(0, 80);
  return normalized || "documento";
}
