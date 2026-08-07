import { $isLinkNode, LinkNode, TOGGLE_LINK_COMMAND } from "@lexical/link";
import {
  $isListNode,
  INSERT_ORDERED_LIST_COMMAND,
  INSERT_UNORDERED_LIST_COMMAND,
  ListItemNode,
  ListNode,
  REMOVE_LIST_COMMAND,
} from "@lexical/list";
import { LexicalComposer } from "@lexical/react/LexicalComposer";
import { useLexicalComposerContext } from "@lexical/react/LexicalComposerContext";
import { ContentEditable } from "@lexical/react/LexicalContentEditable";
import { LexicalErrorBoundary } from "@lexical/react/LexicalErrorBoundary";
import { HistoryPlugin } from "@lexical/react/LexicalHistoryPlugin";
import { LinkPlugin } from "@lexical/react/LexicalLinkPlugin";
import { ListPlugin } from "@lexical/react/LexicalListPlugin";
import { MarkdownShortcutPlugin } from "@lexical/react/LexicalMarkdownShortcutPlugin";
import { OnChangePlugin } from "@lexical/react/LexicalOnChangePlugin";
import { RichTextPlugin } from "@lexical/react/LexicalRichTextPlugin";
import {
  BOLD_ITALIC_STAR,
  BOLD_STAR,
  HEADING,
  ITALIC_STAR,
  LINK,
  ORDERED_LIST,
  QUOTE,
  UNORDERED_LIST,
} from "@lexical/markdown";
import {
  $createHeadingNode,
  $createQuoteNode,
  $isHeadingNode,
  $isQuoteNode,
  HeadingNode,
  QuoteNode,
} from "@lexical/rich-text";
import { $setBlocksType } from "@lexical/selection";
import {
  $createParagraphNode,
  $getSelection,
  $isRangeSelection,
  CAN_REDO_COMMAND,
  CAN_UNDO_COMMAND,
  COMMAND_PRIORITY_LOW,
  FORMAT_TEXT_COMMAND,
  REDO_COMMAND,
  UNDO_COMMAND,
  type LexicalNode,
  type SerializedEditorState,
} from "lexical";
import { useCallback, useEffect, useMemo, useState } from "react";

import {
  EditorToolbar,
  type BlockStyle,
  type TextFormat,
} from "./EditorToolbar";
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
      nodes: [HeadingNode, QuoteNode, ListNode, ListItemNode, LinkNode],
      editorState: editorState ? JSON.stringify(editorState) : undefined,
      theme: {
        heading: { h2: styles.heading },
        link: styles.link,
        list: {
          ol: styles.orderedList,
          ul: styles.list,
          listitem: styles.listItem,
        },
        paragraph: styles.paragraph,
        quote: styles.quote,
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
        <FormattingControls />
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
          <HistoryPlugin />
          <ListPlugin />
          <LinkPlugin validateUrl={isAllowedLink} />
          {/* Typing "## " or "- " becomes the block, so writing does not
              require leaving the keyboard for the toolbar. Only the marks this
              editor actually supports are listed — code and tables are not
              part of a rich text element. */}
          <MarkdownShortcutPlugin transformers={markdownShortcuts} />
        </div>
      </div>
    </LexicalComposer>
  );
}

const markdownShortcuts = [
  HEADING,
  QUOTE,
  UNORDERED_LIST,
  ORDERED_LIST,
  BOLD_ITALIC_STAR,
  BOLD_STAR,
  ITALIC_STAR,
  LINK,
];

const emptyFormat: TextFormat = {
  bold: false,
  italic: false,
  underline: false,
  bulletList: false,
  numberedList: false,
  link: false,
  block: "paragraph",
};

function FormattingControls() {
  const [editor] = useLexicalComposerContext();
  const [format, setFormat] = useState<TextFormat>(emptyFormat);
  const [history, setHistory] = useState({ canUndo: false, canRedo: false });
  const [showLinkInput, setShowLinkInput] = useState(false);
  const [link, setLink] = useState("");

  useEffect(
    () =>
      editor.registerUpdateListener(({ editorState }) => {
        editorState.read(() => {
          const selection = $getSelection();
          if (!$isRangeSelection(selection)) return;
          const parents = selectionParents(selection.anchor.getNode());
          const list = parents.find($isListNode);
          setFormat({
            bold: selection.hasFormat("bold"),
            italic: selection.hasFormat("italic"),
            underline: selection.hasFormat("underline"),
            bulletList: list?.getListType() === "bullet",
            numberedList: list?.getListType() === "number",
            link: parents.some($isLinkNode),
            block: parents.some($isHeadingNode)
              ? "heading"
              : parents.some($isQuoteNode)
                ? "quote"
                : "paragraph",
          });
        });
      }),
    [editor],
  );

  useEffect(() => {
    const stopUndo = editor.registerCommand(
      CAN_UNDO_COMMAND,
      (canUndo: boolean) => {
        setHistory((current) => ({ ...current, canUndo }));
        return false;
      },
      COMMAND_PRIORITY_LOW,
    );
    const stopRedo = editor.registerCommand(
      CAN_REDO_COMMAND,
      (canRedo: boolean) => {
        setHistory((current) => ({ ...current, canRedo }));
        return false;
      },
      COMMAND_PRIORITY_LOW,
    );
    return () => {
      stopUndo();
      stopRedo();
    };
  }, [editor]);

  function setBlock(style: BlockStyle) {
    editor.update(() => {
      const selection = $getSelection();
      if (!$isRangeSelection(selection)) return;
      $setBlocksType(selection, () => {
        if (style === "heading") return $createHeadingNode("h2");
        if (style === "quote") return $createQuoteNode();
        return $createParagraphNode();
      });
    });
  }

  function toggleList(type: "bullet" | "number") {
    const active = type === "bullet" ? format.bulletList : format.numberedList;
    if (active) {
      editor.dispatchCommand(REMOVE_LIST_COMMAND, undefined);
      return;
    }
    editor.dispatchCommand(
      type === "bullet"
        ? INSERT_UNORDERED_LIST_COMMAND
        : INSERT_ORDERED_LIST_COMMAND,
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
      <EditorToolbar
        format={format}
        canUndo={history.canUndo}
        canRedo={history.canRedo}
        actions={{
          undo: () => editor.dispatchCommand(UNDO_COMMAND, undefined),
          redo: () => editor.dispatchCommand(REDO_COMMAND, undefined),
          setBlock,
          toggleBold: () => editor.dispatchCommand(FORMAT_TEXT_COMMAND, "bold"),
          toggleItalic: () =>
            editor.dispatchCommand(FORMAT_TEXT_COMMAND, "italic"),
          toggleUnderline: () =>
            editor.dispatchCommand(FORMAT_TEXT_COMMAND, "underline"),
          toggleBulletList: () => toggleList("bullet"),
          toggleNumberedList: () => toggleList("number"),
          toggleLink,
        }}
      />
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
