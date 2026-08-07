import type { DocumentContent, DocumentSection } from "../../shared/api/client";

/**
 * Reading the outline's state out of the document itself.
 *
 * Which sections are required is a property of the script, not of the text, so
 * this is where a published, versioned template will plug in. Until then the
 * required keys live here rather than being scattered through the components.
 */
const optionalDefinitionKeys = new Set(["conclusao"]);

export type SectionStatus = {
  section: DocumentSection;
  position: number;
  required: boolean;
  words: number;
  filled: boolean;
};

export function sectionStatus(
  content: DocumentContent,
  section: DocumentSection,
): SectionStatus {
  const words = countWords(section);
  return {
    section,
    position: content.sections.findIndex((item) => item.id === section.id) + 1,
    required: !optionalDefinitionKeys.has(section.definitionKey),
    words,
    filled: words > 0,
  };
}

export function documentProgress(content: DocumentContent) {
  const statuses = content.sections.map((section) =>
    sectionStatus(content, section),
  );
  const filled = statuses.filter((status) => status.filled).length;
  const missing = statuses.filter(
    (status) => status.required && !status.filled,
  );
  return {
    statuses,
    percent: statuses.length ? Math.round((filled / statuses.length) * 100) : 0,
    missingRequired: missing.map((status) => status.section.title),
  };
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
