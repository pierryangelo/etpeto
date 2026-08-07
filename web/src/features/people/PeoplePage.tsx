import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { ApiError, api, type Person } from "../../shared/api/client";
import { AppHeader } from "../../shared/ui/AppHeader";
import { DataTable } from "../../shared/ui/DataTable";
import { PageShell } from "../../shared/ui/PageShell";
import {
  Badge,
  Button,
  CopyableValue,
  Modal,
  PageHeading,
  Select,
  TextField,
} from "../../shared/ui/controls";

const roleLabels: Record<string, string> = {
  organization_admin: "Administração",
  template_manager: "Gestor de roteiros",
  author: "Autor",
  installation_admin: "Instalação",
};

const assignableRoles = [
  { value: "author", label: roleLabels.author! },
  { value: "template_manager", label: roleLabels.template_manager! },
  { value: "organization_admin", label: roleLabels.organization_admin! },
];

export function PeoplePage() {
  const queryClient = useQueryClient();
  const [inviting, setInviting] = useState(false);
  const [issuedLink, setIssuedLink] = useState("");
  const [error, setError] = useState("");

  const people = useQuery({
    queryKey: ["people"],
    queryFn: () => api.listPeople(),
  });

  const refresh = () => queryClient.invalidateQueries({ queryKey: ["people"] });

  const revoke = useMutation({
    mutationFn: (id: string) => api.revokeInvitation(id),
    onSuccess: refresh,
    onError: (failure: unknown) => setError(describe(failure)),
  });

  const changeRole = useMutation({
    mutationFn: (person: Person & { next: string }) =>
      api.changePersonRole(person.userId ?? "", person.role, person.next),
    onSuccess: refresh,
    onError: (failure: unknown) => setError(describe(failure)),
  });

  return (
    <PageShell header={<AppHeader />}>
      <PageHeading
        eyebrow="Acesso"
        title="Pessoas do órgão"
        description="O papel define o que cada pessoa pode fazer. Ser administração não concede autoria sobre os documentos das outras pessoas."
        action={
          <Button
            variant="primary"
            onClick={() => {
              setError("");
              setIssuedLink("");
              setInviting(true);
            }}
          >
            Convidar pessoa
          </Button>
        }
      />

      {error && <p role="alert">{error}</p>}

      {people.isPending && <p role="status">Carregando pessoas…</p>}
      {people.isError && (
        <p role="alert">
          Não foi possível carregar as pessoas.{" "}
          <Button small onClick={() => void people.refetch()}>
            Tentar novamente
          </Button>
        </p>
      )}

      {people.isSuccess && (
        <DataTable
          columns={["Pessoa", "E-mail", "Papel", "Situação", ""]}
          rows={people.data.map((person) => ({
            key: person.userId ?? person.invitationId ?? person.email,
            cells: [
              { content: person.displayName, emphasis: true },
              { content: person.email },
              {
                content: person.pending ? (
                  (roleLabels[person.role] ?? person.role)
                ) : (
                  <Select
                    label={`Papel de ${person.displayName}`}
                    value={person.role}
                    options={assignableRoles}
                    onChange={(next) => changeRole.mutate({ ...person, next })}
                  />
                ),
              },
              {
                content: person.pending ? (
                  <Badge tone="draft">Convite pendente</Badge>
                ) : (
                  <Badge tone="live">Com acesso</Badge>
                ),
              },
              {
                align: "right",
                content: person.pending ? (
                  <Button
                    small
                    variant="danger"
                    onClick={() => revoke.mutate(person.invitationId ?? "")}
                  >
                    Revogar convite
                  </Button>
                ) : null,
              },
            ],
          }))}
        />
      )}

      <InviteModal
        open={inviting}
        issuedLink={issuedLink}
        onClose={() => setInviting(false)}
        onIssued={(link) => {
          setIssuedLink(link);
          void refresh();
        }}
      />
    </PageShell>
  );
}

function InviteModal({
  open,
  issuedLink,
  onClose,
  onIssued,
}: {
  open: boolean;
  issuedLink: string;
  onClose: () => void;
  onIssued: (link: string) => void;
}) {
  const [email, setEmail] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [role, setRole] = useState("author");

  const [failure, setFailure] = useState("");

  const invite = useMutation({
    mutationFn: () => api.invitePerson(email, displayName, role),
    onSuccess: (result) => {
      // The token is shown here and never again: it is not stored in clear and
      // cannot be retrieved. Losing it means issuing a new invitation.
      onIssued(`${window.location.origin}/convite?token=${result.inviteToken}`);
      setEmail("");
      setDisplayName("");
      setFailure("");
    },
    onError: (error: unknown) => setFailure(describe(error)),
  });

  return (
    <Modal
      open={open}
      onOpenChange={(next) => !next && onClose()}
      title={issuedLink ? "Convite criado" : "Convidar pessoa"}
      description={
        issuedLink
          ? "Envie este link pelo canal do órgão. Ele serve uma única vez, vale por 72 horas e não poderá ser consultado de novo."
          : "A pessoa define a própria senha ao aceitar. A administração nunca a conhece."
      }
    >
      {issuedLink ? (
        <div className="ds-stack">
          <CopyableValue value={issuedLink} label="Link do convite" />
          <div className="flex justify-end">
            <Button variant="primary" onClick={onClose}>
              Concluir
            </Button>
          </div>
        </div>
      ) : (
        <form
          className="ds-stack"
          onSubmit={(event) => {
            event.preventDefault();
            setFailure("");
            invite.mutate();
          }}
        >
          {failure && <p role="alert">{failure}</p>}
          <TextField
            label="Nome"
            value={displayName}
            onChange={setDisplayName}
            maxLength={200}
          />
          <TextField
            label="E-mail institucional"
            type="email"
            value={email}
            onChange={setEmail}
            maxLength={320}
          />
          <Select
            label="Papel"
            value={role}
            options={assignableRoles}
            onChange={setRole}
          />
          <div className="flex justify-end gap-1.5">
            <Button variant="quiet" onClick={onClose}>
              Cancelar
            </Button>
            <Button type="submit" variant="primary" disabled={invite.isPending}>
              {invite.isPending ? "Criando…" : "Criar convite"}
            </Button>
          </div>
        </form>
      )}
    </Modal>
  );
}

function describe(failure: unknown): string {
  return failure instanceof ApiError
    ? failure.message
    : "Não foi possível concluir a ação.";
}
