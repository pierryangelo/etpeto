import { useRef, useState, type FormEvent } from "react";
import { z } from "zod";

import styles from "./LoginPage.module.css";

const credentialsSchema = z.object({
  email: z
    .string()
    .trim()
    .email("Informe um e-mail institucional válido.")
    .max(320),
  password: z
    .string()
    .min(1, "Informe sua senha.")
    .max(1024, "A senha informada é muito longa."),
});

export type LoginCredentials = z.infer<typeof credentialsSchema>;

type LoginPageProps = {
  login?: (credentials: LoginCredentials) => Promise<void>;
  confirmSession?: () => Promise<void>;
  navigate?: (path: string) => void;
};

export function LoginPage({
  login = requestLogin,
  confirmSession = requestSessionConfirmation,
  navigate = navigateBrowser,
}: LoginPageProps) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const submittingRef = useRef(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submittingRef.current) return;
    setError("");
    const parsed = credentialsSchema.safeParse({ email, password });
    if (!parsed.success) {
      setError(
        parsed.error.issues[0]?.message ?? "Revise os dados informados.",
      );
      return;
    }

    submittingRef.current = true;
    setSubmitting(true);
    try {
      await login(parsed.data);
      await confirmSession();
      navigate("/documentos");
    } catch {
      setError(
        "Não foi possível entrar. Confira seus dados e tente novamente.",
      );
    } finally {
      submittingRef.current = false;
      setSubmitting(false);
    }
  }

  return (
    <main className={styles.page}>
      <section className={styles.context} aria-label="Sobre o Etpeto">
        <a className={styles.product} href="/" aria-label="Etpeto — início">
          <span className={styles.productMark} aria-hidden="true">
            E
          </span>
          Etpeto
        </a>
        <div className={styles.contextCopy}>
          <span className={styles.eyebrow}>
            Documentos públicos bem estruturados
          </span>
          <h1>Do roteiro institucional ao PDF, sem brigar com a formatação.</h1>
          <p>
            Preencha o conteúdo. O Etpeto preserva a estrutura, o histórico e a
            apresentação oficial do documento.
          </p>
        </div>
        <p className={styles.openSource}>Software livre · AGPL-3.0</p>
      </section>

      <section className={styles.formRegion} aria-labelledby="login-title">
        <form className={styles.card} onSubmit={submit} noValidate>
          <div className={styles.cardHeading}>
            <span className={styles.eyebrow}>Acesso à organização</span>
            <h2 id="login-title">Entre no Etpeto</h2>
            <p>Use as credenciais fornecidas pelo seu órgão.</p>
          </div>

          {error && (
            <div className={styles.error} role="alert">
              {error}
            </div>
          )}

          <label className={styles.field}>
            <span>E-mail institucional</span>
            <input
              type="email"
              name="email"
              autoComplete="username"
              value={email}
              disabled={submitting}
              onChange={(event) => setEmail(event.target.value)}
            />
          </label>

          <label className={styles.field}>
            <span>Senha</span>
            <input
              type="password"
              name="password"
              autoComplete="current-password"
              value={password}
              disabled={submitting}
              onChange={(event) => setPassword(event.target.value)}
            />
          </label>

          <button className={styles.submit} type="submit" disabled={submitting}>
            {submitting ? "Entrando…" : "Entrar"}
          </button>
          <p className={styles.help}>
            Problemas de acesso? Procure a administração do Etpeto no seu órgão.
          </p>
        </form>
      </section>
    </main>
  );
}

async function requestLogin(credentials: LoginCredentials) {
  const response = await fetch("/api/v1/auth/local/login", {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(credentials),
  });
  if (!response.ok) throw new Error("login failed");
}

async function requestSessionConfirmation() {
  const response = await fetch("/api/v1/auth/session", {
    credentials: "include",
    headers: { Accept: "application/json" },
  });
  if (!response.ok) throw new Error("session confirmation failed");
}

function navigateBrowser(path: string) {
  window.location.assign(path);
}
