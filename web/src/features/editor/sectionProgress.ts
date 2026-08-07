import type { DocumentContent, DocumentSection } from "../../shared/api/client";

/**
 * Reading the outline's state out of the document itself.
 *
 * A subsection is a section nested inside another one, so everything here walks
 * the tree in reading order and derives the number from the position. Numbering
 * is never stored: moving a section renumbers the document instead of leaving a
 * stale label behind. Obligation comes from the document, which carries what
 * the script declared, section by section.
 */

export type FlatSection = {
  section: DocumentSection;
  /** Hierarchical number derived from the position, such as "2.1.1". */
  number: string;
  /** 1 for a top level section, 2 for a subsection, and so on. */
  level: number;
  /** Identifiers from the root down, used to hide a collapsed branch. */
  path: string[];
  parentId?: string;
};

export type SectionStatus = FlatSection & {
  required: boolean;
  words: number;
  filled: boolean;
};

export function flattenSections(sections: DocumentSection[]): FlatSection[] {
  const flattened: FlatSection[] = [];
  walk(sections, "", 1, [], undefined, flattened);
  return flattened;
}

function walk(
  sections: DocumentSection[],
  prefix: string,
  level: number,
  ancestors: string[],
  parentId: string | undefined,
  into: FlatSection[],
) {
  sections.forEach((section, index) => {
    const number = prefix ? `${prefix}.${index + 1}` : String(index + 1);
    const path = [...ancestors, section.id];
    into.push({ section, number, level, path, parentId });
    walk(section.sections ?? [], number, level + 1, path, section.id, into);
  });
}

function describe(entry: FlatSection): SectionStatus {
  const words = countWords(entry.section);
  return {
    ...entry,
    required: entry.section.required,
    words,
    filled: words > 0,
  };
}

export function sectionStatus(
  content: DocumentContent,
  section: DocumentSection,
): SectionStatus {
  const entry = flattenSections(content.sections).find(
    (candidate) => candidate.section.id === section.id,
  );
  return describe(
    entry ?? { section, number: "1", level: 1, path: [section.id] },
  );
}

export function documentProgress(content: DocumentContent) {
  const statuses = flattenSections(content.sections).map(describe);
  const filled = statuses.filter((status) => status.filled).length;
  // A pending section is any required one still empty, at any level: a required
  // parent does not vouch for its children, nor the other way round.
  const missing = statuses.filter(
    (status) => status.required && !status.filled,
  );
  return {
    statuses,
    percent: statuses.length ? Math.round((filled / statuses.length) * 100) : 0,
    missingRequired: missing.map((status) => status.section.title),
  };
}

/** Whether a section holds content of its own, ignoring its subsections. */
export function sectionHasContent(section: DocumentSection): boolean {
  return countWords(section) > 0;
}

/**
 * Counts words in the stored editor state rather than in the DOM, so the number
 * matches what will be persisted and rendered — including sections that are not
 * currently open in the editor.
 */
export function countWords(section: DocumentSection): number {
  return section.elements.reduce(
    (total, element) => total + countStateWords(element.richText),
    0,
  );
}

function countStateWords(state: unknown): number {
  const text = collectText(state);
  const words = text.trim().match(/\S+/g);
  return words ? words.length : 0;
}

function collectText(node: unknown): string {
  if (!isRecord(node)) return "";
  let text = typeof node.text === "string" ? `${node.text} ` : "";
  if (isRecord(node.root)) text += collectText(node.root);
  if (Array.isArray(node.children)) {
    for (const child of node.children) text += collectText(child);
  }
  return text;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}
