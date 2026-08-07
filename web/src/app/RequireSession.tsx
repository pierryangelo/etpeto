import { Navigate, Outlet } from "react-router";

import { isUnauthenticated, useSession } from "../features/auth/session";

/**
 * Gate for private routes. The session lives in a cookie the browser cannot
 * read, so the only way to know whether it is valid is to ask the server.
 */
export function RequireSession() {
  const session = useSession();

  if (session.isPending) {
    return (
      <p role="status" style={{ padding: "var(--space-6)" }}>
        Verificando sessão…
      </p>
    );
  }
  if (session.isError && isUnauthenticated(session.error)) {
    return <Navigate to="/entrar" replace />;
  }
  if (session.isError) {
    return (
      <p role="alert" style={{ padding: "var(--space-6)" }}>
        Não foi possível confirmar sua sessão. Verifique a conexão e recarregue
        a página.
      </p>
    );
  }
  return <Outlet />;
}
