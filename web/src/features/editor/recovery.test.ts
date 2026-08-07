import { describe, expect, it } from "vitest";

import { createRecoveryStore } from "./recovery";

describe("createRecoveryStore", () => {
  it("persiste por documento em uma chave versionada", () => {
    const storage = new StorageStub();
    const store = createRecoveryStore(storage, () => "2026-08-06T12:00:00Z");

    store.save("documento-1", 4, { text: "rascunho" });

    expect(storage.keys()).toEqual(["etpeto:recovery:v1:documento-1"]);
    expect(store.load("documento-1")).toEqual({
      schemaVersion: 1,
      documentId: "documento-1",
      serverRevision: 4,
      savedAt: "2026-08-06T12:00:00Z",
      content: { text: "rascunho" },
    });
  });

  it("não deixa uma confirmação antiga apagar revisão local mais nova", () => {
    const storage = new StorageStub();
    const store = createRecoveryStore(storage);
    store.save("documento-1", 8, { text: "mais novo" });

    expect(store.clearPersistedRevision("documento-1", 7)).toBe(false);
    expect(store.load("documento-1")?.serverRevision).toBe(8);
    expect(store.clearPersistedRevision("documento-1", 8)).toBe(true);
    expect(store.load("documento-1")).toBeNull();
  });

  it("ignora JSON inválido sem impedir a edição", () => {
    const storage = new StorageStub();
    storage.setItem("etpeto:recovery:v1:documento-1", "{incompleto");

    expect(createRecoveryStore(storage).load("documento-1")).toBeNull();
  });
});

class StorageStub implements Storage {
  readonly #values = new Map<string, string>();

  get length() {
    return this.#values.size;
  }

  clear() {
    this.#values.clear();
  }

  getItem(key: string) {
    return this.#values.get(key) ?? null;
  }

  key(index: number) {
    return [...this.#values.keys()][index] ?? null;
  }

  removeItem(key: string) {
    this.#values.delete(key);
  }

  setItem(key: string, value: string) {
    this.#values.set(key, value);
  }

  keys() {
    return [...this.#values.keys()];
  }
}
