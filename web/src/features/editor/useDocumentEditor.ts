import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { SerializedEditorState } from "lexical";

import {
  ApiError,
  api,
  type DocumentContent,
  type StoredDocument,
} from "../../shared/api/client";

export type SaveState =
  "idle" | "dirty" | "saving" | "saved" | "conflict" | "failed";

const autosaveDelayMs = 1200;

/**
 * Owns one document: what the server has, what the person changed, and which
 * revision the pending save is based on. Nothing is written back without an
 * expected revision, so a save can never silently overwrite another session.
 */
export function useDocumentEditor(documentId: string) {
  const queryClient = useQueryClient();
  const [draft, setDraft] = useState<DocumentContent | null>(null);
  const [saveState, setSaveState] = useState<SaveState>("idle");
  const revisionRef = useRef(0);
  const pendingRef = useRef<DocumentContent | null>(null);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const document = useQuery({
    queryKey: ["document", documentId],
    queryFn: () => api.openDocument(documentId),
    retry: (failureCount, error) => {
      if (error instanceof ApiError && error.status < 500) return false;
      return failureCount < 2;
    },
  });

  useEffect(() => {
    if (!document.data) return;
    revisionRef.current = document.data.revision;
    setDraft((current) => current ?? document.data.content);
  }, [document.data]);

  const acceptStored = useCallback(
    (stored: StoredDocument) => {
      revisionRef.current = stored.revision;
      queryClient.setQueryData(["document", documentId], stored);
      void queryClient.invalidateQueries({ queryKey: ["documents"] });
    },
    [documentId, queryClient],
  );

  const save = useMutation({
    mutationFn: (content: DocumentContent) =>
      api.saveDocument(documentId, revisionRef.current, content),
    onSuccess: (stored) => {
      acceptStored(stored);
      setSaveState(pendingRef.current ? "dirty" : "saved");
    },
    onError: (error: unknown) => {
      setSaveState(
        error instanceof ApiError && error.isConflict ? "conflict" : "failed",
      );
    },
  });

  const flush = useCallback(() => {
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
    const pending = pendingRef.current;
    if (!pending || save.isPending) return;
    pendingRef.current = null;
    setSaveState("saving");
    save.mutate(pending);
  }, [save]);

  const scheduleSave = useCallback(
    (content: DocumentContent) => {
      pendingRef.current = content;
      setSaveState("dirty");
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(flush, autosaveDelayMs);
    },
    [flush],
  );

  const updateElement = useCallback(
    (sectionId: string, elementId: string, state: SerializedEditorState) => {
      setDraft((current) => {
        if (!current) return current;
        const next: DocumentContent = {
          ...current,
          sections: current.sections.map((section) =>
            section.id !== sectionId
              ? section
              : {
                  ...section,
                  elements: section.elements.map((element) =>
                    element.id !== elementId
                      ? element
                      : { ...element, richText: state as never },
                  ),
                },
          ),
        };
        scheduleSave(next);
        return next;
      });
    },
    [scheduleSave],
  );

  // A pending change must not be lost when the tab closes or the person
  // navigates away, so the debounce is bypassed on unmount.
  useEffect(() => flush, [flush]);

  const reloadRemote = useCallback(async () => {
    pendingRef.current = null;
    const stored = await api.openDocument(documentId);
    acceptStored(stored);
    setDraft(stored.content);
    setSaveState("saved");
  }, [acceptStored, documentId]);

  const keepLocal = useCallback(async () => {
    const stored = await api.openDocument(documentId);
    revisionRef.current = stored.revision;
    queryClient.setQueryData(["document", documentId], stored);
    if (draft) {
      pendingRef.current = draft;
      flush();
    }
  }, [documentId, draft, flush, queryClient]);

  return useMemo(
    () => ({
      draft,
      revision: document.data?.revision ?? 0,
      isLoading: document.isPending,
      loadError: document.error,
      saveState,
      updateElement,
      flush,
      reloadRemote,
      keepLocal,
    }),
    [
      draft,
      document.data?.revision,
      document.isPending,
      document.error,
      flush,
      keepLocal,
      reloadRemote,
      saveState,
      updateElement,
    ],
  );
}
