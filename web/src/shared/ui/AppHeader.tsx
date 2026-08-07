import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "react-router";

import { api } from "../api/client";
import { useSession } from "../../features/auth/session";
import styles from "./AppHeader.module.css";

type AppHeaderProps = {
  children?: React.ReactNode;
};

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
