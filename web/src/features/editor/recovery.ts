import { z } from "zod";

const recoveryPrefix = "etpeto:recovery:v1:";

const recoveryEnvelopeSchema = z.object({
  schemaVersion: z.literal(1),
  documentId: z.string().min(1),
  serverRevision: z.number().int().nonnegative(),
  savedAt: z.string().min(1),
  content: z.unknown(),
});

export type RecoveryEnvelope<TContent = unknown> = Omit<
  z.infer<typeof recoveryEnvelopeSchema>,
  "content"
> & {
  content: TContent;
};

export function createRecoveryStore<TContent = unknown>(
  storage: Storage,
  now: () => string = () => new Date().toISOString(),
) {
  function key(documentId: string) {
    return `${recoveryPrefix}${documentId}`;
  }

  function load(documentId: string): RecoveryEnvelope<TContent> | null {
    try {
      const serialized = storage.getItem(key(documentId));
      if (!serialized) return null;
      const parsed = recoveryEnvelopeSchema.safeParse(JSON.parse(serialized));
      if (!parsed.success || parsed.data.documentId !== documentId) return null;
      return parsed.data as RecoveryEnvelope<TContent>;
    } catch {
      return null;
    }
  }

  function save(
    documentId: string,
    serverRevision: number,
    content: TContent,
  ): RecoveryEnvelope<TContent> {
    const envelope: RecoveryEnvelope<TContent> = {
      schemaVersion: 1,
      documentId,
      serverRevision,
      savedAt: now(),
      content,
    };
    storage.setItem(key(documentId), JSON.stringify(envelope));
    return envelope;
  }

  function clearPersistedRevision(
    documentId: string,
    acknowledgedRevision: number,
  ): boolean {
    const persisted = load(documentId);
    if (!persisted || acknowledgedRevision < persisted.serverRevision) {
      return false;
    }
    storage.removeItem(key(documentId));
    return true;
  }

  return { load, save, clearPersistedRevision };
}
