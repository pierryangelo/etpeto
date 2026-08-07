/**
 * Panel geometry for the editor workspace.
 *
 * Widths are per device, never part of the document: the same ETP opened on
 * another machine keeps its content and takes that machine's layout.
 */

export const layoutStorageKey = "etpeto:editor-layout:v2";
export const legacyLayoutStorageKey = "etpeto:editor-layout:v1";

// Below the minimum a panel stops being usable — the outline truncates every
// section title, and the preview no longer shows a full page width.
export const outlineBounds = { min: 200, max: 420, default: 232 } as const;
export const previewBounds = { min: 240, max: 560, default: 300 } as const;

export type PanelLayout = {
  outlineWidth: number;
  previewWidth: number;
  outlineOpen: boolean;
  previewOpen: boolean;
};

export function defaultLayout(): PanelLayout {
  return {
    outlineWidth: outlineBounds.default,
    previewWidth: previewBounds.default,
    outlineOpen: true,
    previewOpen: true,
  };
}

export function clampOutline(width: number): number {
  return clamp(width, outlineBounds.min, outlineBounds.max);
}

export function clampPreview(width: number): number {
  return clamp(width, previewBounds.min, previewBounds.max);
}

export function readLayout(storage: Storage): PanelLayout {
  const stored = parse(storage, layoutStorageKey);
  if (isLayout(stored)) {
    return {
      outlineWidth: clampOutline(stored.outlineWidth),
      previewWidth: clampPreview(stored.previewWidth),
      outlineOpen: stored.outlineOpen,
      previewOpen: stored.previewOpen,
    };
  }

  // A device that only ever stored the collapsed state keeps it.
  const legacy = parse(storage, legacyLayoutStorageKey);
  if (isCollapseOnly(legacy)) {
    return {
      ...defaultLayout(),
      outlineOpen: legacy.outlineOpen,
      previewOpen: legacy.previewOpen,
    };
  }
  return defaultLayout();
}

export function writeLayout(storage: Storage, layout: PanelLayout): void {
  try {
    storage.setItem(layoutStorageKey, JSON.stringify(layout));
  } catch {
    // Layout is an enhancement. A full or blocked storage must never stop
    // someone from editing a document.
  }
}

function clamp(value: number, minimum: number, maximum: number): number {
  if (!Number.isFinite(value)) return minimum;
  return Math.round(Math.min(Math.max(value, minimum), maximum));
}

function parse(storage: Storage, key: string): unknown {
  try {
    const raw = storage.getItem(key);
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

function isLayout(value: unknown): value is PanelLayout {
  return (
    isRecord(value) &&
    typeof value.outlineWidth === "number" &&
    typeof value.previewWidth === "number" &&
    typeof value.outlineOpen === "boolean" &&
    typeof value.previewOpen === "boolean"
  );
}

function isCollapseOnly(
  value: unknown,
): value is { outlineOpen: boolean; previewOpen: boolean } {
  return (
    isRecord(value) &&
    typeof value.outlineOpen === "boolean" &&
    typeof value.previewOpen === "boolean"
  );
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}
