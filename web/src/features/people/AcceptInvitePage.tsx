import { useMutation, useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { useNavigate, useSearchParams } from "react-router";

import { ApiError, api } from "../../shared/api/client";
import { Button, PageHeading, TextField } from "../../shared/ui/controls";
import { PageShell } from "../../shared/ui/PageShell";

const minimumPasswordLength = 12;

/**
 * Where an invitation becomes an account.
 *
 * Reached without a session — it is how someone with no account gets one. The
 * page only ever shows the name and address the administration already wrote on
 * the invitation, and the password is chosen here, by the person.
 */
export function AcceptInvitePage() {
  const [params] = useSearchParams();
  const navigate = useNavigate();
  const token = params.get("token") ?? "";
  const [password, setPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [error, setError] = useState("");

  const invitation = useQuery({
    queryKey: ["invitation", token],
    queryFn: () => api.describeInvitation(token),
    enabled: token !== "",
    retry: false,
  });

  const accept = useMutation({
    mutationFn: () => api.acceptInvitation(token, password),
    onSuccess: () => navigate("/entrar"),
    onError: (failure: unknown) => {
      setError(
        failure instanceof ApiError
          ? failure.message
          : "Não foi possível concluir o convite.",
      );
    },
  });

  if (!token || invitation.isError) {
    return (
      <PageShell header={null}>
        <PageHeading
          eyebrow="Convite"
          title="Este convite não pode ser usado"
          description="O link pode ter expirado, já ter sido utilizado ou estar incompleto. Peça um novo à administração do seu órgão."
        />
      </PageShell>
    );
  }

  return (
    <PageShell header={null}>
      <PageHeading
        eyebrow="Convite"
        title={
          invitation.data
            ? `Bem-vindo, ${invitation.data.displayName}`
            : "Confirmando convite…"
        }
        description={
          invitation.data
            ? `Defina a senha de acesso para ${invitation.data.email}. Ninguém além de você a conhecerá.`
            : undefined
        }
      />

      {error && <p role="alert">{error}</p>}

      <form
        className="ds-stack max-w-md"
        onSubmit={(event) => {
          event.preventDefault();
          setError("");
          if (password.length < minimumPasswordLength) {
            setError(
              `A senha precisa ter ao menos ${minimumPasswordLength} caracteres.`,
            );
            return;
          }
          if (password !== confirmation) {
            setError("As duas senhas não coincidem.");
            return;
          }
          accept.mutate();
        }}
      >
        <TextField
          label="Senha"
          type="password"
          autoComplete="new-password"
          value={password}
          onChange={setPassword}
          help={`Ao menos ${minimumPasswordLength} caracteres.`}
        />
        <TextField
          label="Repita a senha"
          type="password"
          autoComplete="new-password"
          value={confirmation}
          onChange={setConfirmation}
        />
        <div>
          <Button
            type="submit"
            variant="primary"
            disabled={accept.isPending || invitation.isPending}
          >
            {accept.isPending ? "Concluindo…" : "Definir senha e entrar"}
          </Button>
        </div>
      </form>
    </PageShell>
  );
}
