import { Collapsible } from "@base-ui/react/collapsible";
import { useCallback, useMemo, useRef, useState } from "react";

import type { DocumentSection } from "../../shared/api/client";
import { flattenSections, sectionHasContent } from "./sectionProgress";

/**
 * The outline is a tree, so it is exposed as one.
 *
 * Base UI has Accordion and Collapsible but no tree, and the two are not the
 * same contract: a tree carries level, position and set size, and moves with
 * the arrow keys as a single tab stop. Collapsible still does the opening and
 * closing; what is written here is the part Base UI does not cover.
 */

type OutlineTreeProps = {
  sections: DocumentSection[];
  activeSectionId: string;
  onSelect: (id: string) => void;
};

export function OutlineTree({
  sections,
  activeSectionId,
  onSelect,
}: OutlineTreeProps) {
  const [collapsed, setCollapsed] = useState<ReadonlySet<string>>(
    () => new Set(),
  );
  const treeRef = useRef<HTMLUListElement>(null);

  // Only the sections currently on screen take part in arrow navigation: a
  // collapsed branch hides its children from the keyboard too.
  const visible = useMemo(
    () =>
      flattenSections(sections).filter(
        (entry) => !isHidden(entry.path, collapsed),
      ),
    [collapsed, sections],
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
        ?.querySelector<HTMLElement>(`[data-section-id="${target.section.id}"]`)
        ?.focus();
    },
    [visible],
  );

  const handleKeyDown = useCallback(
    (event: React.KeyboardEvent<HTMLLIElement>, index: number) => {
      const entry = visible[index];
      if (!entry) return;
      const hasChildren = (entry.section.sections?.length ?? 0) > 0;
      const open = hasChildren && !collapsed.has(entry.section.id);

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
          else toggle(entry.section.id, true);
          break;
        case "ArrowLeft":
          if (open) {
            toggle(entry.section.id, false);
            break;
          }
          // Already closed, or a leaf: go up to the parent.
          focusAt(
            visible.findIndex(
              (candidate) => candidate.section.id === entry.parentId,
            ),
          );
          break;
        case "Home":
          focusAt(0);
          break;
        case "End":
          focusAt(visible.length - 1);
          break;
        case "Enter":
        case " ":
          onSelect(entry.section.id);
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
      aria-label="Roteiro do documento"
      className="m-0 list-none p-1"
    >
      {sections.map((section, index) => (
        <OutlineBranch
          key={section.id}
          section={section}
          number={String(index + 1)}
          level={1}
          position={index + 1}
          setSize={sections.length}
          activeSectionId={activeSectionId}
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

type BranchProps = {
  section: DocumentSection;
  number: string;
  level: number;
  position: number;
  setSize: number;
  activeSectionId: string;
  collapsed: ReadonlySet<string>;
  visible: ReturnType<typeof flattenSections>;
  onToggle: (id: string, open: boolean) => void;
  onSelect: (id: string) => void;
  onKeyDown: (event: React.KeyboardEvent<HTMLLIElement>, index: number) => void;
};

function OutlineBranch({
  section,
  number,
  level,
  position,
  setSize,
  activeSectionId,
  collapsed,
  visible,
  onToggle,
  onSelect,
  onKeyDown,
}: BranchProps) {
  const children = section.sections ?? [];
  const hasChildren = children.length > 0;
  const open = hasChildren && !collapsed.has(section.id);
  const active = section.id === activeSectionId;
  const index = visible.findIndex((entry) => entry.section.id === section.id);
  const filled = sectionHasContent(section);
  const labelId = `outline-label-${section.id}`;

  const item = (
    <li
      role="treeitem"
      data-section-id={section.id}
      // The group of children lives inside this element, so the accessible
      // name has to be pinned to the section's own label. Without this the
      // name of a branch would swallow every descendant's title.
      aria-labelledby={labelId}
      aria-level={level}
      aria-posinset={position}
      aria-setsize={setSize}
      aria-selected={active}
      aria-expanded={hasChildren ? open : undefined}
      // A single tab stop: Tab reaches the section being edited, and the arrow
      // keys move from there.
      tabIndex={active ? 0 : -1}
      onKeyDown={(event) => onKeyDown(event, index)}
      onClick={(event) => {
        event.stopPropagation();
        onSelect(section.id);
      }}
      className="cursor-pointer rounded-1 outline-none"
      data-active={active || undefined}
    >
      <div
        className="grid grid-cols-[auto_18px_minmax(0,1fr)_auto] items-center gap-2 rounded-1 px-2 py-1.5 text-12.5px hover:bg-paper data-[active]:bg-paper data-[active]:font-600 data-[active]:text-ink data-[active]:shadow-[inset_2px_0_0_var(--color-accent)]"
        data-active={active || undefined}
        style={{ paddingLeft: `${(level - 1) * 14 + 8}px` }}
      >
        {hasChildren ? (
          <Collapsible.Trigger
            className="grid size-4 place-items-center rounded-0.5 text-subtle hover:bg-line hover:text-ink"
            aria-label={`${open ? "Recolher" : "Expandir"} ${section.title}`}
            // The tree owns a single tab stop, and the arrow keys already open
            // and close a branch. A focusable trigger here would put one stop
            // per section back into the tab order.
            tabIndex={-1}
            onClick={(event) => {
              event.stopPropagation();
              onToggle(section.id, !open);
            }}
          >
            <span
              className={
                open ? "i-lucide-chevron-down" : "i-lucide-chevron-right"
              }
            />
          </Collapsible.Trigger>
        ) : (
          <span className="size-4" aria-hidden="true" />
        )}

        <span
          className="grid size-4.5 place-items-center rounded-1 bg-line text-10px font-700 tabular-nums text-subtle data-[filled]:bg-success-soft data-[filled]:text-success"
          aria-hidden="true"
          data-filled={filled || undefined}
        >
          {filled ? "✓" : ""}
        </span>

        <span id={labelId} className="flex items-baseline gap-1.5 truncate">
          <span className="shrink-0 text-10.5px font-600 tabular-nums text-subtle">
            {number}
          </span>
          <span className="truncate">{section.title}</span>
        </span>

        <span
          className="text-9.5px font-700 uppercase tracking-wider text-subtle data-[pending]:text-accent-strong"
          data-pending={section.required && !filled ? "" : undefined}
        >
          {section.required ? "Obrig." : "Opcional"}
        </span>
      </div>

      {hasChildren && (
        <Collapsible.Panel keepMounted={false}>
          <ul role="group" className="m-0 list-none p-0">
            {children.map((child, childIndex) => (
              <OutlineBranch
                key={child.id}
                section={child}
                number={`${number}.${childIndex + 1}`}
                level={level + 1}
                position={childIndex + 1}
                setSize={children.length}
                activeSectionId={activeSectionId}
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
      onOpenChange={(next) => onToggle(section.id, next)}
      render={item}
    />
  );
}

function isHidden(path: string[], collapsed: ReadonlySet<string>): boolean {
  return path.slice(0, -1).some((ancestor) => collapsed.has(ancestor));
}
