import { Separator } from "@base-ui/react/separator";
import { useCallback, useRef } from "react";

import styles from "./EditorWorkspace.module.css";

type PanelDividerProps = {
  label: string;
  width: number;
  min: number;
  max: number;
  /** Which way a rightward drag grows the panel. */
  grow: "right" | "left";
  onResize: (width: number) => void;
  onReset: () => void;
};

const keyboardStep = 16;

/**
 * The 1px line between panels is also the drag handle. It stays a separator for
 * assistive technology, and carries the range so the width is announced and can
 * be changed from the keyboard — dragging must never be the only way.
 */
export function PanelDivider({
  label,
  width,
  min,
  max,
  grow,
  onResize,
  onReset,
}: PanelDividerProps) {
  // The drag is tracked here rather than through pointer capture state, so the
  // behaviour does not depend on an API the environment may not implement.
  const drag = useRef<{ pointerId: number; x: number; width: number } | null>(
    null,
  );

  const handlePointerDown = useCallback(
    (event: React.PointerEvent<HTMLDivElement>) => {
      if (event.button !== 0) return;
      event.preventDefault();
      drag.current = { pointerId: event.pointerId, x: event.clientX, width };
      event.currentTarget.setPointerCapture?.(event.pointerId);
    },
    [width],
  );

  const handlePointerMove = useCallback(
    (event: React.PointerEvent<HTMLDivElement>) => {
      const active = drag.current;
      if (!active || active.pointerId !== event.pointerId) return;
      const travelled = event.clientX - active.x;
      onResize(active.width + (grow === "right" ? travelled : -travelled));
    },
    [grow, onResize],
  );

  const handlePointerUp = useCallback(
    (event: React.PointerEvent<HTMLDivElement>) => {
      if (drag.current?.pointerId !== event.pointerId) return;
      drag.current = null;
      event.currentTarget.releasePointerCapture?.(event.pointerId);
    },
    [],
  );

  const handleKeyDown = useCallback(
    (event: React.KeyboardEvent<HTMLDivElement>) => {
      const towardsStart = grow === "right" ? -keyboardStep : keyboardStep;
      const towardsEnd = -towardsStart;
      switch (event.key) {
        case "ArrowLeft":
          onResize(width + towardsStart);
          break;
        case "ArrowRight":
          onResize(width + towardsEnd);
          break;
        case "Home":
          onResize(min);
          break;
        case "End":
          onResize(max);
          break;
        case "Enter":
          onReset();
          break;
        default:
          return;
      }
      event.preventDefault();
    },
    [grow, max, min, onReset, onResize, width],
  );

  return (
    <Separator
      orientation="vertical"
      className={styles.divider}
      role="separator"
      aria-label={label}
      aria-orientation="vertical"
      aria-valuenow={width}
      aria-valuemin={min}
      aria-valuemax={max}
      tabIndex={0}
      onPointerDown={handlePointerDown}
      onPointerMove={handlePointerMove}
      onPointerUp={handlePointerUp}
      onDoubleClick={onReset}
      onKeyDown={handleKeyDown}
    >
      <span className={styles.grip} aria-hidden="true" />
    </Separator>
  );
}
