import { Menu } from "@base-ui/react/menu";
import { Toolbar } from "@base-ui/react/toolbar";
import { Tooltip } from "@base-ui/react/tooltip";
import type { ReactNode } from "react";

// Compiled into the bundle at build time; no request ever leaves the page.
import IconBold from "~icons/lucide/bold";
import IconChevronDown from "~icons/lucide/chevron-down";
import IconItalic from "~icons/lucide/italic";
import IconLink from "~icons/lucide/link";
import IconList from "~icons/lucide/list";
import IconListOrdered from "~icons/lucide/list-ordered";
import IconRedo from "~icons/lucide/redo-2";
import IconUnderline from "~icons/lucide/underline";
import IconUndo from "~icons/lucide/undo-2";
import IconUnlink from "~icons/lucide/unlink";

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
          <IconUndo />
        </Tool>
        <Tool
          label="Refazer"
          shortcut={`${modifier}⇧Z`}
          disabled={!canRedo}
          onClick={actions.redo}
        >
          <IconRedo />
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
          <IconBold />
        </Tool>
        <Tool
          label="Itálico"
          shortcut={`${modifier}I`}
          pressed={format.italic}
          onClick={actions.toggleItalic}
        >
          <IconItalic />
        </Tool>
        <Tool
          label="Sublinhado"
          shortcut={`${modifier}U`}
          pressed={format.underline}
          onClick={actions.toggleUnderline}
        >
          <IconUnderline />
        </Tool>

        <Toolbar.Separator className={styles.separator} />

        <Tool
          label="Lista com marcadores"
          pressed={format.bulletList}
          onClick={actions.toggleBulletList}
        >
          <IconList />
        </Tool>
        <Tool
          label="Lista numerada"
          pressed={format.numberedList}
          onClick={actions.toggleNumberedList}
        >
          <IconListOrdered />
        </Tool>

        <Toolbar.Separator className={styles.separator} />

        <Tool
          label={format.link ? "Remover link" : "Inserir link"}
          shortcut={`${modifier}K`}
          pressed={format.link}
          onClick={actions.toggleLink}
        >
          {format.link ? <IconUnlink /> : <IconLink />}
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
        <IconChevronDown className={styles.caret} />
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
