import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Link, NavLink, useNavigate } from "react-router";

import { api } from "../api/client";
import { useSession } from "../../features/auth/session";
import styles from "./AppHeader.module.css";

type AppHeaderProps = {
  children?: React.ReactNode;
};

const scriptManagerRoles = new Set([
  "organization_admin",
  "template_manager",
  "installation_admin",
]);

function canManageScripts(role: string | undefined): boolean {
  return role !== undefined && scriptManagerRoles.has(role);
}

export function AppHeader({ children }: AppHeaderProps) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const session = useSession();

  const logout = useMutation({
    mutationFn: () => api.logout(),
    onSettled: async () => {
      queryClient.clear();
      navigate("/entrar");
    },
  });

  return (
    <header className={styles.header}>
      <Link
        className={styles.product}
        to="/documentos"
        aria-label="Etpeto — documentos"
      >
        <span className={styles.productMark} aria-hidden="true">
          E
        </span>
        <span className={styles.productName}>Etpeto</span>
      </Link>
      {/* Only the areas the role can actually use are offered. The API checks
          the role again on every request; this is presentation, not control. */}
      <nav className={styles.nav} aria-label="Áreas do Etpeto">
        <NavLink to="/documentos">Documentos</NavLink>
        {canManageScripts(session.data?.role) && (
          <NavLink to="/roteiros">Roteiros</NavLink>
        )}
      </nav>
      {children}
      <div className={styles.account}>
        {session.data && (
          <span className={styles.userName}>{session.data.displayName}</span>
        )}
        <button
          className={styles.logout}
          type="button"
          disabled={logout.isPending}
          onClick={() => logout.mutate()}
        >
          Sair
        </button>
      </div>
    </header>
  );
}
