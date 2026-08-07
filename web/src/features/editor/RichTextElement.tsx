import { $isLinkNode, LinkNode, TOGGLE_LINK_COMMAND } from "@lexical/link";
import {
  $isListNode,
  INSERT_UNORDERED_LIST_COMMAND,
  ListItemNode,
  ListNode,
  REMOVE_LIST_COMMAND,
} from "@lexical/list";
import { LexicalComposer } from "@lexical/react/LexicalComposer";
import { useLexicalComposerContext } from "@lexical/react/LexicalComposerContext";
import { ContentEditable } from "@lexical/react/LexicalContentEditable";
import { LexicalErrorBoundary } from "@lexical/react/LexicalErrorBoundary";
import { LinkPlugin } from "@lexical/react/LexicalLinkPlugin";
import { ListPlugin } from "@lexical/react/LexicalListPlugin";
import { OnChangePlugin } from "@lexical/react/LexicalOnChangePlugin";
import { RichTextPlugin } from "@lexical/react/LexicalRichTextPlugin";
import {
  $createHeadingNode,
  $isHeadingNode,
  HeadingNode,
} from "@lexical/rich-text";
import { $setBlocksType } from "@lexical/selection";
import {
  $createParagraphNode,
  $getSelection,
  $isRangeSelection,
  FORMAT_TEXT_COMMAND,
  type LexicalNode,
  type SerializedEditorState,
} from "lexical";
import { useCallback, useEffect, useMemo, useState } from "react";

import { createRecoveryStore } from "./recovery";
import styles from "./RichTextElement.module.css";

type RichTextElementProps = {
  documentId: string;
  elementId: string;
  initialState?: SerializedEditorState;
  onChange: (state: SerializedEditorState) => void;
};

type RecoveryContent = {
  elements: Record<string, SerializedEditorState>;
};

export function RichTextElement({
  documentId,
  elementId,
  initialState,
  onChange,
}: RichTextElementProps) {
  const recoveredState = useMemo(
    () => loadRecoveredState(documentId, elementId),
    [documentId, elementId],
  );
  const editorState = recoveredState ?? initialState;
  const initialConfig = useMemo(
    () => ({
      namespace: `etpeto:${documentId}:${elementId}`,
      nodes: [HeadingNode, ListNode, ListItemNode, LinkNode],
      editorState: editorState ? JSON.stringify(editorState) : undefined,
      theme: {
        heading: { h2: styles.heading },
        link: styles.link,
        list: { ul: styles.list, listitem: styles.listItem },
        paragraph: styles.paragraph,
        text: {
          bold: styles.bold,
          italic: styles.italic,
          underline: styles.underline,
        },
      },
      onError(error: Error) {
        throw error;
      },
    }),
    [documentId, editorState, elementId],
  );

  const handleChange = useCallback(
    (nextState: { toJSON(): SerializedEditorState }) => {
      const serialized = nextState.toJSON();
      onChange(serialized);
      saveRecoveredState(documentId, elementId, serialized);
    },
    [documentId, elementId, onChange],
  );

  return (
    <LexicalComposer initialConfig={initialConfig}>
      <div className={styles.container}>
        <Toolbar />
        <div className={styles.surface}>
          <RichTextPlugin
            contentEditable={
              <ContentEditable
                className={styles.contentEditable}
                aria-label="Conteúdo da seção"
              />
            }
            placeholder={
              <div className={styles.placeholder}>
                Comece a descrever esta seção…
              </div>
            }
            ErrorBoundary={LexicalErrorBoundary}
          />
          <OnChangePlugin onChange={handleChange} ignoreSelectionChange />
          <ListPlugin />
          <LinkPlugin validateUrl={isAllowedLink} />
        </div>
      </div>
    </LexicalComposer>
  );
}

function Toolbar() {
  const [editor] = useLexicalComposerContext();
  const [format, setFormat] = useState({
    bold: false,
    italic: false,
    underline: false,
    heading: false,
    list: false,
    link: false,
  });
  const [showLinkInput, setShowLinkInput] = useState(false);
  const [link, setLink] = useState("");

  useEffect(
    () =>
      editor.registerUpdateListener(({ editorState }) => {
        editorState.read(() => {
          const selection = $getSelection();
          if (!$isRangeSelection(selection)) return;
          const parents = selectionParents(selection.anchor.getNode());
          setFormat({
            bold: selection.hasFormat("bold"),
            italic: selection.hasFormat("italic"),
            underline: selection.hasFormat("underline"),
            heading: parents.some($isHeadingNode),
            list: parents.some($isListNode),
            link: parents.some($isLinkNode),
          });
        });
      }),
    [editor],
  );

  function toggleHeading() {
    editor.update(() => {
      const selection = $getSelection();
      if (!$isRangeSelection(selection)) return;
      $setBlocksType(selection, () =>
        format.heading ? $createParagraphNode() : $createHeadingNode("h2"),
      );
    });
  }

  function toggleList() {
    editor.dispatchCommand(
      format.list ? REMOVE_LIST_COMMAND : INSERT_UNORDERED_LIST_COMMAND,
      undefined,
    );
  }

  function toggleLink() {
    if (format.link) {
      editor.dispatchCommand(TOGGLE_LINK_COMMAND, null);
      setShowLinkInput(false);
      return;
    }
    setShowLinkInput((visible) => !visible);
  }

  function applyLink() {
    const normalized = normalizeLink(link);
    if (!normalized) return;
    editor.dispatchCommand(TOGGLE_LINK_COMMAND, {
      url: normalized,
      rel: "noreferrer",
      target: "_blank",
    });
    setLink("");
    setShowLinkInput(false);
  }

  return (
    <div className={styles.toolbarArea}>
      <div className={styles.toolbar} role="toolbar" aria-label="Formatação">
        <ToolbarButton
          label="Negrito"
          pressed={format.bold}
          onClick={() => editor.dispatchCommand(FORMAT_TEXT_COMMAND, "bold")}
        >
          <strong>B</strong>
        </ToolbarButton>
        <ToolbarButton
          label="Itálico"
          pressed={format.italic}
          onClick={() => editor.dispatchCommand(FORMAT_TEXT_COMMAND, "italic")}
        >
          <em>I</em>
        </ToolbarButton>
        <ToolbarButton
          label="Sublinhado"
          pressed={format.underline}
          onClick={() =>
            editor.dispatchCommand(FORMAT_TEXT_COMMAND, "underline")
          }
        >
          <span className={styles.underlineGlyph}>U</span>
        </ToolbarButton>
        <span className={styles.separator} aria-hidden="true" />
        <ToolbarButton
          label="Título"
          pressed={format.heading}
          onClick={toggleHeading}
        >
          H2
        </ToolbarButton>
        <ToolbarButton
          label="Lista com marcadores"
          pressed={format.list}
          onClick={toggleList}
        >
          ☷
        </ToolbarButton>
        <span className={styles.separator} aria-hidden="true" />
        <ToolbarButton
          label={format.link ? "Remover link" : "Inserir link"}
          pressed={format.link}
          onClick={toggleLink}
        >
          ↗
        </ToolbarButton>
      </div>
      {showLinkInput && (
        <div className={styles.linkEditor} role="group" aria-label="Novo link">
          <label>
            <span>Endereço</span>
            <input
              type="url"
              value={link}
              placeholder="https://exemplo.gov.br"
              onChange={(event) => setLink(event.target.value)}
            />
          </label>
          <button
            type="button"
            onClick={applyLink}
            disabled={!normalizeLink(link)}
          >
            Aplicar
          </button>
        </div>
      )}
    </div>
  );
}

function ToolbarButton({
  label,
  pressed,
  onClick,
  children,
}: {
  label: string;
  pressed: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      aria-pressed={pressed}
      onMouseDown={(event) => event.preventDefault()}
      onClick={onClick}
    >
      {children}
    </button>
  );
}

function selectionParents(node: LexicalNode): LexicalNode[] {
  const parents: LexicalNode[] = [];
  let current: LexicalNode | null = node;
  while (current) {
    parents.push(current);
    current = current.getParent();
  }
  return parents;
}

function normalizeLink(value: string): string | null {
  const candidate = value.trim();
  if (!candidate) return null;
  const withScheme = /^[a-z][a-z0-9+.-]*:/i.test(candidate)
    ? candidate
    : `https://${candidate}`;
  return isAllowedLink(withScheme) ? withScheme : null;
}

function isAllowedLink(value: string): boolean {
  try {
    const url = new URL(value);
    return url.protocol === "https:" || url.protocol === "http:";
  } catch {
    return false;
  }
}

function loadRecoveredState(
  documentId: string,
  elementId: string,
): SerializedEditorState | undefined {
  if (typeof localStorage === "undefined") return undefined;
  const persisted =
    createRecoveryStore<RecoveryContent>(localStorage).load(documentId);
  if (
    isRecord(persisted?.content) &&
    isRecord(persisted.content.elements) &&
    isSerializedEditorState(persisted.content.elements[elementId])
  ) {
    return persisted.content.elements[elementId];
  }
  return undefined;
}

function saveRecoveredState(
  documentId: string,
  elementId: string,
  state: SerializedEditorState,
) {
  if (typeof localStorage === "undefined") return;
  try {
    const store = createRecoveryStore<RecoveryContent>(localStorage);
    const persisted = store.load(documentId);
    const elements = isRecord(persisted?.content?.elements)
      ? persisted.content.elements
      : {};
    store.save(documentId, persisted?.serverRevision ?? 0, {
      elements: { ...elements, [elementId]: state },
    });
  } catch {
    // Local recovery is best effort; storage failure must not block editing.
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isSerializedEditorState(
  value: unknown,
): value is SerializedEditorState {
  return isRecord(value) && "root" in value && isRecord(value.root);
}
