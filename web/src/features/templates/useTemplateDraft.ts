import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";

import {
  ApiError,
  api,
  type Blueprint,
  type BlueprintSection,
} from "../../shared/api/client";

export type DraftState =
  "idle" | "dirty" | "saving" | "saved" | "conflict" | "failed";

const autosaveDelayMs = 1200;

/**
 * Owns the working draft of a script.
 *
 * Editing here never reaches an author: the draft is private to whoever manages
 * the script, and only publishing creates the revision documents are written
 * from. The draft version travels with every write, so a second manager editing
 * the same script produces a conflict instead of a silent overwrite.
 */
export function useTemplateDraft(templateId: string) {
  const queryClient = useQueryClient();
  const [blueprint, setBlueprint] = useState<Blueprint | null>(null);
  const [state, setState] = useState<DraftState>("idle");
  const blueprintRef = useRef<Blueprint | null>(null);
  const versionRef = useRef(0);
  const pendingRef = useRef<Blueprint | null>(null);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const template = useQuery({
    queryKey: ["template", templateId],
    queryFn: () => api.openTemplate(templateId),
    retry: (failureCount, error) => {
      if (error instanceof ApiError && error.status < 500) return false;
      return failureCount < 2;
    },
  });

  useEffect(() => {
    if (!template.data) return;
    versionRef.current = template.data.draftVersion;
    if (blueprintRef.current) return;
    blueprintRef.current = template.data.blueprint;
    setBlueprint(template.data.blueprint);
  }, [template.data]);

  const save = useMutation({
    mutationFn: (next: Blueprint) =>
      api.saveTemplateDraft(templateId, versionRef.current, next),
    onSuccess: (saved) => {
      versionRef.current = saved.draftVersion;
      queryClient.setQueryData(["template", templateId], saved);
      void queryClient.invalidateQueries({ queryKey: ["templates"] });
      setState(pendingRef.current ? "dirty" : "saved");
    },
    onError: (error: unknown) => {
      setState(
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
    setState("saving");
    save.mutate(pending);
  }, [save]);

  // Reached through a ref so the cleanup runs on unmount only, and not on every
  // re-render that changes the callback's identity.
  const flushRef = useRef(flush);
  useEffect(() => {
    flushRef.current = flush;
  }, [flush]);
  useEffect(() => () => flushRef.current(), []);

  const apply = useCallback(
    (transform: (sections: BlueprintSection[]) => BlueprintSection[]) => {
      const current = blueprintRef.current;
      if (!current) return;
      const next: Blueprint = {
        ...current,
        sections: transform(current.sections),
      };
      if (next.sections === current.sections) return;
      blueprintRef.current = next;
      setBlueprint(next);
      pendingRef.current = next;
      setState("dirty");
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(flush, autosaveDelayMs);
    },
    [flush],
  );

  const publish = useMutation({
    mutationFn: async () => {
      flush();
      return api.publishTemplate(templateId);
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ["template", templateId],
      });
      await queryClient.invalidateQueries({ queryKey: ["templates"] });
    },
  });

  const discard = useCallback(async () => {
    pendingRef.current = null;
    const reloaded = await api.openTemplate(templateId);
    versionRef.current = reloaded.draftVersion;
    blueprintRef.current = reloaded.blueprint;
    setBlueprint(reloaded.blueprint);
    queryClient.setQueryData(["template", templateId], reloaded);
    setState("saved");
  }, [queryClient, templateId]);

  return {
    template: template.data,
    blueprint,
    isLoading: template.isPending,
    loadError: template.error,
    state,
    apply,
    flush,
    discard,
    publish,
  };
}
