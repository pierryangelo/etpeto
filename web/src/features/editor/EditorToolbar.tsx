import { Menu } from "@base-ui/react/menu";
import { Toolbar } from "@base-ui/react/toolbar";
import { Tooltip } from "@base-ui/react/tooltip";
import type { ReactNode } from "react";

import styles from "./RichTextElement.module.css";

export type BlockStyle = "paragraph" | "heading" | "quote";

export type TextFormat = {
  bold: boolean;
  italic: boolean;
  underline: boolean;
  bulletList: boolean;
  numberedList: boolean;
  link: boolean;
  block: BlockStyle;
};

export type ToolbarActions = {
  undo: () => void;
  redo: () => void;
  setBlock: (style: BlockStyle) => void;
  toggleBold: () => void;
  toggleItalic: () => void;
  toggleUnderline: () => void;
  toggleBulletList: () => void;
  toggleNumberedList: () => void;
  toggleLink: () => void;
};

const blockLabels: Record<BlockStyle, string> = {
  paragraph: "Parágrafo",
  heading: "Título",
  quote: "Citação",
};

// The platform modifier is shown as written on the person's own keyboard.
const modifier =
  typeof navigator !== "undefined" && /Mac|iPhone|iPad/.test(navigator.platform)
    ? "⌘"
    : "Ctrl+";

export function EditorToolbar({
  format,
  actions,
  canUndo,
  canRedo,
}: {
  format: TextFormat;
  actions: ToolbarActions;
  canUndo: boolean;
  canRedo: boolean;
}) {
  return (
    <Tooltip.Provider delay={400}>
      <Toolbar.Root className={styles.toolbar} aria-label="Formatação">
        <Tool
          label="Desfazer"
          shortcut={`${modifier}Z`}
          disabled={!canUndo}
          onClick={actions.undo}
        >
          ↶
        </Tool>
        <Tool
          label="Refazer"
          shortcut={`${modifier}⇧Z`}
          disabled={!canRedo}
          onClick={actions.redo}
        >
          ↷
        </Tool>

        <Toolbar.Separator className={styles.separator} />

        <BlockMenu current={format.block} onSelect={actions.setBlock} />

        <Toolbar.Separator className={styles.separator} />

        <Tool
          label="Negrito"
          shortcut={`${modifier}B`}
          pressed={format.bold}
          onClick={actions.toggleBold}
        >
          <b>B</b>
        </Tool>
        <Tool
          label="Itálico"
          shortcut={`${modifier}I`}
          pressed={format.italic}
          onClick={actions.toggleItalic}
        >
          <i>I</i>
        </Tool>
        <Tool
          label="Sublinhado"
          shortcut={`${modifier}U`}
          pressed={format.underline}
          onClick={actions.toggleUnderline}
        >
          <span className={styles.underlineGlyph}>U</span>
        </Tool>

        <Toolbar.Separator className={styles.separator} />

        <Tool
          label="Lista com marcadores"
          pressed={format.bulletList}
          onClick={actions.toggleBulletList}
        >
          •—
        </Tool>
        <Tool
          label="Lista numerada"
          pressed={format.numberedList}
          onClick={actions.toggleNumberedList}
        >
          1—
        </Tool>

        <Toolbar.Separator className={styles.separator} />

        <Tool
          label={format.link ? "Remover link" : "Inserir link"}
          shortcut={`${modifier}K`}
          pressed={format.link}
          onClick={actions.toggleLink}
        >
          ↗
        </Tool>
      </Toolbar.Root>
    </Tooltip.Provider>
  );
}

function Tool({
  label,
  shortcut,
  pressed,
  disabled,
  onClick,
  children,
}: {
  label: string;
  shortcut?: string;
  pressed?: boolean;
  disabled?: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <Tooltip.Root>
      <Tooltip.Trigger
        render={
          <Toolbar.Button
            className={styles.tool}
            aria-label={label}
            aria-pressed={pressed}
            disabled={disabled}
            // Keeping the mouse from stealing focus preserves the selection the
            // command is about to act on.
            onMouseDown={(event) => event.preventDefault()}
            onClick={onClick}
          />
        }
      >
        {children}
      </Tooltip.Trigger>
      <Tooltip.Portal>
        <Tooltip.Positioner side="bottom" sideOffset={6}>
          <Tooltip.Popup className={styles.tooltip}>
            {label}
            {shortcut && <kbd>{shortcut}</kbd>}
          </Tooltip.Popup>
        </Tooltip.Positioner>
      </Tooltip.Portal>
    </Tooltip.Root>
  );
}

function BlockMenu({
  current,
  onSelect,
}: {
  current: BlockStyle;
  onSelect: (style: BlockStyle) => void;
}) {
  return (
    <Menu.Root>
      <Menu.Trigger
        render={
          <Toolbar.Button
            className={`${styles.tool} ${styles.blockTrigger}`}
            aria-label="Estilo do bloco"
            onMouseDown={(event) => event.preventDefault()}
          />
        }
      >
        {blockLabels[current]}
        <span aria-hidden="true">▾</span>
      </Menu.Trigger>
      <Menu.Portal>
        <Menu.Positioner side="bottom" align="start" sideOffset={6}>
          <Menu.Popup className={styles.menu}>
            {(Object.keys(blockLabels) as BlockStyle[]).map((style) => (
              <Menu.Item
                key={style}
                className={styles.menuItem}
                data-selected={style === current || undefined}
                onClick={() => onSelect(style)}
              >
                {blockLabels[style]}
              </Menu.Item>
            ))}
          </Menu.Popup>
        </Menu.Positioner>
      </Menu.Portal>
    </Menu.Root>
  );
}
