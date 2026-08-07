import type { BlueprintSection } from "../../shared/api/client";

/**
 * Structural edits on the script tree.
 *
 * Every function returns a new tree and never mutates the argument, so an edit
 * is a value the caller can compare, store or discard. Depth and sibling rules
 * live here rather than in the component: what may be indented is a property of
 * the script, not of the panel drawing it.
 */

export const maxDepth = 4;

export type SectionPatch = Partial<Omit<BlueprintSection, "sections">>;

export function newSection(title = "Nova seção"): BlueprintSection {
  return {
    key: "",
    title,
    guidance: "",
    required: false,
    allowRename: false,
    allowedElements: ["richText"],
  };
}

/**
 * Derives a stable key from the title, unique across the whole script. The key
 * is what a document carries as its definitionKey, so it must survive renaming
 * — it is minted once, when the section is created.
 */
export function deriveKey(title: string, taken: ReadonlySet<string>): string {
  const base =
    title
      .normalize("NFD")
      .replace(/[̀-ͯ]/g, "")
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-+|-+$/g, "")
      .slice(0, 60) || "secao";
  if (!taken.has(base)) return base;
  for (let suffix = 2; suffix < 1000; suffix += 1) {
    const candidate = `${base}-${suffix}`;
    if (!taken.has(candidate)) return candidate;
  }
  return `${base}-${taken.size + 1}`;
}

export function collectKeys(sections: BlueprintSection[]): Set<string> {
  const keys = new Set<string>();
  walk(sections, (section) => keys.add(section.key));
  return keys;
}

export function findSection(
  sections: BlueprintSection[],
  key: string,
): BlueprintSection | undefined {
  let found: BlueprintSection | undefined;
  walk(sections, (section) => {
    if (section.key === key) found = section;
  });
  return found;
}

export function depthOf(sections: BlueprintSection[], key: string): number {
  let depth = 0;
  const visit = (list: BlueprintSection[], level: number) => {
    for (const section of list) {
      if (section.key === key) depth = level;
      visit(section.sections ?? [], level + 1);
    }
  };
  visit(sections, 1);
  return depth;
}

/** Deepest level below a section, counting the section itself as 1. */
export function subtreeHeight(section: BlueprintSection): number {
  const children = section.sections ?? [];
  if (children.length === 0) return 1;
  return 1 + Math.max(...children.map(subtreeHeight));
}

export function addSection(
  sections: BlueprintSection[],
  parentKey: string | null,
  section: BlueprintSection,
): BlueprintSection[] {
  const prepared = {
    ...section,
    key: deriveKey(section.title, collectKeys(sections)),
  };
  if (parentKey === null) return [...sections, prepared];
  return mapTree(sections, (candidate) =>
    candidate.key === parentKey
      ? { ...candidate, sections: [...(candidate.sections ?? []), prepared] }
      : candidate,
  );
}

export function updateSection(
  sections: BlueprintSection[],
  key: string,
  patch: SectionPatch,
): BlueprintSection[] {
  return mapTree(sections, (section) =>
    // The key is deliberately excluded: renaming a section must not change what
    // it means for documents already written from it.
    section.key === key ? { ...section, ...patch, key: section.key } : section,
  );
}

export function removeSection(
  sections: BlueprintSection[],
  key: string,
): BlueprintSection[] {
  return sections
    .filter((section) => section.key !== key)
    .map((section) =>
      withChildren(section, removeSection(section.sections ?? [], key)),
    );
}

export function moveSection(
  sections: BlueprintSection[],
  key: string,
  direction: "up" | "down",
): BlueprintSection[] {
  const index = sections.findIndex((section) => section.key === key);
  if (index >= 0) {
    const target = direction === "up" ? index - 1 : index + 1;
    if (target < 0 || target >= sections.length) return sections;
    const reordered = [...sections];
    const [moved] = reordered.splice(index, 1);
    reordered.splice(target, 0, moved!);
    return reordered;
  }
  return sections.map((section) =>
    withChildren(section, moveSection(section.sections ?? [], key, direction)),
  );
}

/**
 * Turns a section into a subsection of the sibling above it. Refused when there
 * is no sibling above, or when the move would push the subtree past the depth
 * a numbered heading can still carry.
 */
export function indentSection(
  sections: BlueprintSection[],
  key: string,
): BlueprintSection[] {
  const currentDepth = depthOf(sections, key);
  const section = findSection(sections, key);
  if (!section || currentDepth === 0) return sections;
  if (currentDepth + subtreeHeight(section) > maxDepth) return sections;
  return indentWithin(sections, key);
}

function indentWithin(
  sections: BlueprintSection[],
  key: string,
): BlueprintSection[] {
  const index = sections.findIndex((section) => section.key === key);
  if (index > 0) {
    const moved = sections[index]!;
    const previous = sections[index - 1]!;
    const reordered = [...sections];
    reordered.splice(index, 1);
    reordered[index - 1] = {
      ...previous,
      sections: [...(previous.sections ?? []), moved],
    };
    return reordered;
  }
  if (index === 0) return sections;
  return sections.map((section) =>
    withChildren(section, indentWithin(section.sections ?? [], key)),
  );
}

/** Moves a subsection up one level, placing it right after its former parent. */
export function outdentSection(
  sections: BlueprintSection[],
  key: string,
): BlueprintSection[] {
  return sections.flatMap((section) => {
    const children = section.sections ?? [];
    const index = children.findIndex((child) => child.key === key);
    if (index >= 0) {
      const moved = children[index]!;
      const remaining = [...children];
      remaining.splice(index, 1);
      return [withChildren(section, remaining), moved];
    }
    return [withChildren(section, outdentSection(children, key))];
  });
}

export function canIndent(sections: BlueprintSection[], key: string): boolean {
  return indentSection(sections, key) !== sections;
}

export function canOutdent(sections: BlueprintSection[], key: string): boolean {
  return depthOf(sections, key) > 1;
}

function withChildren(
  section: BlueprintSection,
  children: BlueprintSection[],
): BlueprintSection {
  if (children.length === 0) {
    const { sections: _dropped, ...rest } = section;
    return rest;
  }
  return { ...section, sections: children };
}

function mapTree(
  sections: BlueprintSection[],
  transform: (section: BlueprintSection) => BlueprintSection,
): BlueprintSection[] {
  return sections.map((section) => {
    const transformed = transform(section);
    const children = transformed.sections ?? [];
    return children.length > 0
      ? { ...transformed, sections: mapTree(children, transform) }
      : transformed;
  });
}

function walk(
  sections: BlueprintSection[],
  visit: (section: BlueprintSection) => void,
) {
  for (const section of sections) {
    visit(section);
    walk(section.sections ?? [], visit);
  }
}
