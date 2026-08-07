import { z } from "zod";

/**
 * Boundary between the browser and the Etpeto API.
 *
 * Every response is parsed with Zod before reaching the application, so a
 * contract change surfaces as a validation error instead of an undefined field
 * deep inside a component. Generating this client from the OpenAPI document,
 * with a CI check against divergence, is still pending.
 */

export const csrfCookieName = "__Host-etpeto_csrf";

export class ApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }

  get isUnauthenticated(): boolean {
    return this.status === 401;
  }

  get isConflict(): boolean {
    return this.status === 409;
  }
}

export const userSchema = z.object({
  id: z.string(),
  displayName: z.string(),
});

export const documentSummarySchema = z.object({
  id: z.string(),
  title: z.string(),
  revision: z.number().int(),
  updatedAt: z.string(),
});

const richTextStateSchema = z.looseObject({ root: z.unknown() });

export const documentElementSchema = z.object({
  id: z.string(),
  type: z.literal("richText"),
  richText: richTextStateSchema.optional(),
});

// A subsection is a section nested inside another one: the same shape, all the
// way down. The recursion is declared with a getter so the schema can refer to
// itself while it is still being defined.
export type DocumentSection = {
  id: string;
  definitionKey: string;
  title: string;
  guidance?: string;
  required: boolean;
  elements: z.infer<typeof documentElementSchema>[];
  sections?: DocumentSection[];
};

export const documentSectionSchema: z.ZodType<DocumentSection> = z.object({
  id: z.string(),
  definitionKey: z.string(),
  title: z.string(),
  guidance: z.string().optional(),
  required: z.boolean(),
  elements: z.array(documentElementSchema),
  get sections() {
    return z.array(documentSectionSchema).optional();
  },
});

export const documentContentSchema = z.object({
  schemaVersion: z.number().int(),
  documentType: z.string(),
  title: z.string(),
  sections: z.array(documentSectionSchema),
});

export const storedDocumentSchema = z.object({
  id: z.string(),
  revision: z.number().int(),
  updatedAt: z.string(),
  content: documentContentSchema,
});

export type AuthenticatedUser = z.infer<typeof userSchema>;
export type DocumentSummary = z.infer<typeof documentSummarySchema>;
export type DocumentContent = z.infer<typeof documentContentSchema>;
export type StoredDocument = z.infer<typeof storedDocumentSchema>;

/**
 * The CSRF token is published in a readable cookie precisely so it can be
 * echoed back in a header. The session cookie stays HttpOnly, so a cross-site
 * page can send the cookie but never this header.
 */
export function readCsrfToken(): string {
  if (typeof document === "undefined") return "";
  const entry = document.cookie
    .split("; ")
    .find((candidate) => candidate.startsWith(`${csrfCookieName}=`));
  return entry
    ? decodeURIComponent(entry.slice(csrfCookieName.length + 1))
    : "";
}

type RequestOptions = {
  method?: string;
  body?: unknown;
  accept?: string;
};

async function request(path: string, options: RequestOptions = {}) {
  const method = options.method ?? "GET";
  const headers: Record<string, string> = {
    Accept: options.accept ?? "application/json",
  };
  if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
  }
  if (method !== "GET" && method !== "HEAD") {
    headers["X-Etpeto-Csrf"] = readCsrfToken();
  }

  const response = await fetch(path, {
    method,
    credentials: "include",
    headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
  });
  if (!response.ok) {
    throw new ApiError(response.status, await safeProblemDetail(response));
  }
  return response;
}

async function safeProblemDetail(response: Response): Promise<string> {
  try {
    const problem: unknown = await response.json();
    if (
      typeof problem === "object" &&
      problem !== null &&
      "detail" in problem &&
      typeof problem.detail === "string"
    ) {
      return problem.detail;
    }
  } catch {
    // A non-JSON error body is not worth surfacing; the status carries meaning.
  }
  return `request failed with status ${response.status}`;
}

async function requestJson<Schema extends z.ZodType>(
  path: string,
  schema: Schema,
  options: RequestOptions = {},
): Promise<z.infer<Schema>> {
  const response = await request(path, options);
  return schema.parse(await response.json());
}

export const api = {
  async session(): Promise<AuthenticatedUser> {
    const body = await requestJson(
      "/api/v1/auth/session",
      z.object({ user: userSchema }),
    );
    return body.user;
  },

  async login(email: string, password: string): Promise<AuthenticatedUser> {
    const body = await requestJson(
      "/api/v1/auth/local/login",
      z.object({ user: userSchema }),
      { method: "POST", body: { email, password } },
    );
    return body.user;
  },

  async logout(): Promise<void> {
    await request("/api/v1/auth/logout", { method: "POST" });
  },

  async listDocuments(): Promise<DocumentSummary[]> {
    const body = await requestJson(
      "/api/v1/documents",
      z.object({ documents: z.array(documentSummarySchema) }),
    );
    return body.documents;
  },

  createDocument(title: string): Promise<StoredDocument> {
    return requestJson("/api/v1/documents", storedDocumentSchema, {
      method: "POST",
      body: { title },
    });
  },

  openDocument(id: string): Promise<StoredDocument> {
    return requestJson(
      `/api/v1/documents/${encodeURIComponent(id)}`,
      storedDocumentSchema,
    );
  },

  saveDocument(
    id: string,
    expectedRevision: number,
    content: DocumentContent,
  ): Promise<StoredDocument> {
    return requestJson(
      `/api/v1/documents/${encodeURIComponent(id)}`,
      storedDocumentSchema,
      { method: "PUT", body: { expectedRevision, content } },
    );
  },

  async previewDocument(id: string): Promise<Blob> {
    const response = await request(
      `/api/v1/documents/${encodeURIComponent(id)}/preview`,
      { method: "POST", accept: "application/pdf" },
    );
    return response.blob();
  },
};
