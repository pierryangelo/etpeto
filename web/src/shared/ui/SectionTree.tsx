import { Collapsible } from "@base-ui/react/collapsible";
import { useCallback, useMemo, useRef, useState } from "react";

/**
 * The outline of a document and the structure of a script are the same thing:
 * numbered sections that nest, each carrying a state and an obligation. This is
 * that one component, and both screens project their domain onto it.
 *
 * Base UI has Accordion and Collapsible but no tree, and they are not the same
 * contract: a tree carries level, position and set size, and moves with the
 * arrow keys as a single tab stop. Collapsible still opens and closes; what is
 * written here is the part Base UI does not cover.
 */

export type TreeNode = {
  id: string;
  title: string;
  /** Shown at the end of the row, such as an obligation. */
  badge?: { label: string; attention?: boolean };
  /** Marks a section that already has content. */
  complete?: boolean;
  children?: TreeNode[];
};

type SectionTreeProps = {
  nodes: TreeNode[];
  selectedId: string;
  label: string;
  onSelect: (id: string) => void;
};

type FlatNode = {
  node: TreeNode;
  number: string;
  level: number;
  path: string[];
  parentId?: string;
};

export function flattenNodes(nodes: TreeNode[]): FlatNode[] {
  const flattened: FlatNode[] = [];
  const walk = (
    list: TreeNode[],
    prefix: string,
    level: number,
    ancestors: string[],
    parentId?: string,
  ) => {
    list.forEach((node, index) => {
      const number = prefix ? `${prefix}.${index + 1}` : String(index + 1);
      const path = [...ancestors, node.id];
      flattened.push({ node, number, level, path, parentId });
      walk(node.children ?? [], number, level + 1, path, node.id);
    });
  };
  walk(nodes, "", 1, []);
  return flattened;
}

export function SectionTree({
  nodes,
  selectedId,
  label,
  onSelect,
}: SectionTreeProps) {
  const [collapsed, setCollapsed] = useState<ReadonlySet<string>>(
    () => new Set(),
  );
  const treeRef = useRef<HTMLUListElement>(null);

  // Only what is on screen takes part in arrow navigation: a collapsed branch
  // hides its children from the keyboard too.
  const visible = useMemo(
    () =>
      flattenNodes(nodes).filter(
        (entry) => !entry.path.slice(0, -1).some((id) => collapsed.has(id)),
      ),
    [collapsed, nodes],
  );

  const toggle = useCallback((id: string, open: boolean) => {
    setCollapsed((current) => {
      const next = new Set(current);
      if (open) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const focusAt = useCallback(
    (index: number) => {
      const target = visible[Math.max(0, Math.min(index, visible.length - 1))];
      if (!target) return;
      treeRef.current
        ?.querySelector<HTMLElement>(`[data-node-id="${target.node.id}"]`)
        ?.focus();
    },
    [visible],
  );

  const handleKeyDown = useCallback(
    (event: React.KeyboardEvent<HTMLLIElement>, index: number) => {
      const entry = visible[index];
      if (!entry) return;
      const hasChildren = (entry.node.children?.length ?? 0) > 0;
      const open = hasChildren && !collapsed.has(entry.node.id);

      switch (event.key) {
        case "ArrowDown":
          focusAt(index + 1);
          break;
        case "ArrowUp":
          focusAt(index - 1);
          break;
        case "ArrowRight":
          if (!hasChildren) return;
          if (open) focusAt(index + 1);
          else toggle(entry.node.id, true);
          break;
        case "ArrowLeft":
          if (open) {
            toggle(entry.node.id, false);
            break;
          }
          focusAt(visible.findIndex((item) => item.node.id === entry.parentId));
          break;
        case "Home":
          focusAt(0);
          break;
        case "End":
          focusAt(visible.length - 1);
          break;
        case "Enter":
        case " ":
          onSelect(entry.node.id);
          break;
        default:
          return;
      }
      event.preventDefault();
      event.stopPropagation();
    },
    [collapsed, focusAt, onSelect, toggle, visible],
  );

  return (
    <ul
      ref={treeRef}
      role="tree"
      aria-label={label}
      className="m-0 list-none p-1"
    >
      {nodes.map((node, index) => (
        <Branch
          key={node.id}
          node={node}
          number={String(index + 1)}
          level={1}
          position={index + 1}
          setSize={nodes.length}
          selectedId={selectedId}
          collapsed={collapsed}
          visible={visible}
          onToggle={toggle}
          onSelect={onSelect}
          onKeyDown={handleKeyDown}
        />
      ))}
    </ul>
  );
}

function Branch({
  node,
  number,
  level,
  position,
  setSize,
  selectedId,
  collapsed,
  visible,
  onToggle,
  onSelect,
  onKeyDown,
}: {
  node: TreeNode;
  number: string;
  level: number;
  position: number;
  setSize: number;
  selectedId: string;
  collapsed: ReadonlySet<string>;
  visible: FlatNode[];
  onToggle: (id: string, open: boolean) => void;
  onSelect: (id: string) => void;
  onKeyDown: (event: React.KeyboardEvent<HTMLLIElement>, index: number) => void;
}) {
  const children = node.children ?? [];
  const hasChildren = children.length > 0;
  const open = hasChildren && !collapsed.has(node.id);
  const selected = node.id === selectedId;
  const index = visible.findIndex((entry) => entry.node.id === node.id);
  const labelId = `tree-label-${node.id}`;

  const item = (
    <li
      role="treeitem"
      data-node-id={node.id}
      // The group of children lives inside this element, so the accessible name
      // is pinned to the row's own label; otherwise a branch would be named
      // after every descendant it contains.
      aria-labelledby={labelId}
      aria-level={level}
      aria-posinset={position}
      aria-setsize={setSize}
      aria-selected={selected}
      aria-expanded={hasChildren ? open : undefined}
      // One tab stop for the whole tree: Tab reaches the selected section, and
      // the arrow keys move from there.
      tabIndex={selected ? 0 : -1}
      onKeyDown={(event) => onKeyDown(event, index)}
      onClick={(event) => {
        event.stopPropagation();
        onSelect(node.id);
      }}
      className="cursor-pointer rounded-1 outline-none"
      data-selected={selected || undefined}
    >
      <div
        className="grid grid-cols-[16px_18px_minmax(0,1fr)_auto] items-center gap-2 rounded-1 px-2 py-1.5 text-12.5px text-text hover:bg-paper data-[selected]:bg-paper data-[selected]:font-600 data-[selected]:text-ink data-[selected]:shadow-[inset_2px_0_0_var(--color-accent)]"
        data-selected={selected || undefined}
        style={{ paddingLeft: `${(level - 1) * 14 + 8}px` }}
      >
        {hasChildren ? (
          <Collapsible.Trigger
            className="ds-icon-btn size-4"
            aria-label={`${open ? "Recolher" : "Expandir"} ${node.title}`}
            // The tree owns a single tab stop and the arrow keys already open
            // and close a branch.
            tabIndex={-1}
            onClick={(event) => {
              event.stopPropagation();
              onToggle(node.id, !open);
            }}
          >
            <span
              className={
                open ? "i-lucide-chevron-down" : "i-lucide-chevron-right"
              }
            />
          </Collapsible.Trigger>
        ) : (
          <span aria-hidden="true" />
        )}

        <span
          className="grid size-4.5 place-items-center rounded-1 bg-line text-10px font-700 tabular-nums text-subtle data-[complete]:bg-success-soft data-[complete]:text-success"
          aria-hidden="true"
          data-complete={node.complete || undefined}
        >
          {node.complete ? "✓" : number.split(".").pop()}
        </span>

        <span id={labelId} className="flex items-baseline gap-1.5 truncate">
          <span className="shrink-0 text-10.5px font-600 tabular-nums text-subtle">
            {number}
          </span>
          <span className="truncate">{node.title}</span>
        </span>

        {node.badge && (
          <span
            className="text-9.5px font-700 tracking-wide uppercase text-subtle data-[attention]:text-accent-strong"
            data-attention={node.badge.attention || undefined}
          >
            {node.badge.label}
          </span>
        )}
      </div>

      {hasChildren && (
        <Collapsible.Panel keepMounted={false}>
          <ul role="group" className="m-0 list-none p-0">
            {children.map((child, childIndex) => (
              <Branch
                key={child.id}
                node={child}
                number={`${number}.${childIndex + 1}`}
                level={level + 1}
                position={childIndex + 1}
                setSize={children.length}
                selectedId={selectedId}
                collapsed={collapsed}
                visible={visible}
                onToggle={onToggle}
                onSelect={onSelect}
                onKeyDown={onKeyDown}
              />
            ))}
          </ul>
        </Collapsible.Panel>
      )}
    </li>
  );

  if (!hasChildren) return item;

  return (
    <Collapsible.Root
      open={open}
      onOpenChange={(next) => onToggle(node.id, next)}
      render={item}
    />
  );
}
