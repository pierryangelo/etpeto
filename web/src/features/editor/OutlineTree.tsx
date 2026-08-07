import { useMemo } from "react";

import type { DocumentSection } from "../../shared/api/client";
import { SectionTree, type TreeNode } from "../../shared/ui/SectionTree";
import { sectionHasContent } from "./sectionProgress";

/**
 * Projects the document outline onto the shared section tree. The tree knows
 * nothing about documents; this is the only place that maps one to the other.
 */
export function OutlineTree({
  sections,
  activeSectionId,
  onSelect,
}: {
  sections: DocumentSection[];
  activeSectionId: string;
  onSelect: (id: string) => void;
}) {
  const nodes = useMemo(() => toNodes(sections), [sections]);
  return (
    <SectionTree
      nodes={nodes}
      selectedId={activeSectionId}
      label="Roteiro do documento"
      onSelect={onSelect}
    />
  );
}

function toNodes(sections: DocumentSection[]): TreeNode[] {
  return sections.map((section) => {
    const complete = sectionHasContent(section);
    return {
      id: section.id,
      title: section.title,
      complete,
      badge: {
        label: section.required ? "Obrig." : "Opcional",
        attention: section.required && !complete,
      },
      children: toNodes(section.sections ?? []),
    };
  });
}
