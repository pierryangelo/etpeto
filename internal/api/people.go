package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	authdomain "github.com/etpeto/etpeto/internal/auth"
)

// Only the administration manages access. A script manager shapes documents,
// not who may sign in.
var accessAdminRoles = []string{
	authdomain.RoleOrganizationAdmin,
	authdomain.RoleInstallationAdmin,
}

type PeopleService interface {
	List(ctx context.Context, organizationID string) ([]authdomain.Person, error)
	Invite(
		ctx context.Context,
		organizationID string,
		createdBy string,
		email string,
		displayName string,
		role string,
	) (string, authdomain.Person, error)
	Describe(ctx context.Context, token string) (authdomain.PendingInvitation, error)
	Accept(ctx context.Context, token string, password string) (authdomain.AcceptedInvitation, error)
	Revoke(ctx context.Context, organizationID string, invitationID string) error
	ChangeRole(
		ctx context.Context,
		organizationID string,
		userID string,
		currentRole string,
		role string,
	) error
}

type PeopleListOutput struct {
	Body struct {
		People []authdomain.Person `json:"people"`
	}
}

type InviteInput struct {
	Session string `cookie:"__Host-etpeto_session"`
	CSRF    string `header:"X-Etpeto-Csrf"`
	Body    struct {
		Email       string `json:"email" maxLength:"320"`
		DisplayName string `json:"displayName" maxLength:"200"`
		Role        string `json:"role"`
	}
}

type InviteOutput struct {
	Body struct {
		Person authdomain.Person `json:"person"`
		// InviteToken is returned exactly once, for the administration to send
		// through the organization's own channel. It is never stored in clear
		// and cannot be retrieved again.
		InviteToken string `json:"inviteToken"`
	}
}

type RevokeInviteInput struct {
	Session string `cookie:"__Host-etpeto_session"`
	CSRF    string `header:"X-Etpeto-Csrf"`
	ID      string `path:"id"`
}

type ChangeRoleInput struct {
	Session string `cookie:"__Host-etpeto_session"`
	CSRF    string `header:"X-Etpeto-Csrf"`
	ID      string `path:"id"`
	Body    struct {
		CurrentRole string `json:"currentRole"`
		Role        string `json:"role"`
	}
}

type DescribeInviteInput struct {
	Token string `query:"token"`
}

type DescribeInviteOutput struct {
	Body struct {
		DisplayName string `json:"displayName"`
		Email       string `json:"email"`
	}
}

type AcceptInviteInput struct {
	Body struct {
		Token    string `json:"token"`
		Password string `json:"password" maxLength:"1024"`
	}
}

type EmptyOutput struct {
	Status int
}

func registerPeople(api huma.API, people PeopleService, protected guard) {
	huma.Register(api, huma.Operation{
		OperationID: "list-people",
		Method:      http.MethodGet,
		Path:        "/api/v1/people",
		Summary:     "List who has access to the organization",
		Tags:        []string{"People"},
	}, func(ctx context.Context, input *ListDocumentsInput) (*PeopleListOutput, error) {
		user, err := protected.requireRoles(ctx, input.Session, accessAdminRoles...)
		if err != nil {
			return nil, err
		}
		if people == nil {
			return nil, huma.Error503ServiceUnavailable("Pessoas temporariamente indisponíveis")
		}
		listed, err := people.List(ctx, user.OrganizationID)
		if err != nil {
			return nil, peopleError(err)
		}
		output := &PeopleListOutput{}
		output.Body.People = listed
		if output.Body.People == nil {
			output.Body.People = []authdomain.Person{}
		}
		return output, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "invite-person",
		Method:        http.MethodPost,
		Path:          "/api/v1/people/invitations",
		Summary:       "Create a single-use invitation link",
		Tags:          []string{"People"},
		DefaultStatus: http.StatusCreated,
		MaxBodyBytes:  4 << 10,
	}, func(ctx context.Context, input *InviteInput) (*InviteOutput, error) {
		user, err := protected.authorizeRoles(ctx, input.Session, input.CSRF, accessAdminRoles...)
		if err != nil {
			return nil, err
		}
		if people == nil {
			return nil, huma.Error503ServiceUnavailable("Pessoas temporariamente indisponíveis")
		}
		token, person, err := people.Invite(
			ctx,
			user.OrganizationID,
			user.ID,
			input.Body.Email,
			input.Body.DisplayName,
			input.Body.Role,
		)
		if err != nil {
			return nil, peopleError(err)
		}
		output := &InviteOutput{}
		output.Body.Person = person
		output.Body.InviteToken = token
		return output, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "revoke-invitation",
		Method:        http.MethodDelete,
		Path:          "/api/v1/people/invitations/{id}",
		Summary:       "Revoke a pending invitation",
		Tags:          []string{"People"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *RevokeInviteInput) (*EmptyOutput, error) {
		user, err := protected.authorizeRoles(ctx, input.Session, input.CSRF, accessAdminRoles...)
		if err != nil {
			return nil, err
		}
		if people == nil {
			return nil, huma.Error503ServiceUnavailable("Pessoas temporariamente indisponíveis")
		}
		if err := people.Revoke(ctx, user.OrganizationID, input.ID); err != nil {
			return nil, peopleError(err)
		}
		return &EmptyOutput{Status: http.StatusNoContent}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "change-person-role",
		Method:        http.MethodPut,
		Path:          "/api/v1/people/{id}/role",
		Summary:       "Change the role of a person",
		Tags:          []string{"People"},
		DefaultStatus: http.StatusNoContent,
		MaxBodyBytes:  1 << 10,
	}, func(ctx context.Context, input *ChangeRoleInput) (*EmptyOutput, error) {
		user, err := protected.authorizeRoles(ctx, input.Session, input.CSRF, accessAdminRoles...)
		if err != nil {
			return nil, err
		}
		if people == nil {
			return nil, huma.Error503ServiceUnavailable("Pessoas temporariamente indisponíveis")
		}
		if err := people.ChangeRole(
			ctx, user.OrganizationID, input.ID, input.Body.CurrentRole, input.Body.Role,
		); err != nil {
			return nil, peopleError(err)
		}
		return &EmptyOutput{Status: http.StatusNoContent}, nil
	})

	// The two routes below are reached without a session: they are how someone
	// who has no account yet gets one. They expose only the name and address
	// already written on the invitation by the administration.
	huma.Register(api, huma.Operation{
		OperationID: "describe-invitation",
		Method:      http.MethodGet,
		Path:        "/api/v1/invitations",
		Summary:     "Describe who a pending invitation is for",
		Tags:        []string{"People"},
	}, func(ctx context.Context, input *DescribeInviteInput) (*DescribeInviteOutput, error) {
		if people == nil {
			return nil, huma.Error503ServiceUnavailable("Convites temporariamente indisponíveis")
		}
		invitation, err := people.Describe(ctx, input.Token)
		if err != nil {
			return nil, peopleError(err)
		}
		output := &DescribeInviteOutput{}
		output.Body.DisplayName = invitation.DisplayName
		output.Body.Email = invitation.Email
		return output, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "accept-invitation",
		Method:        http.MethodPost,
		Path:          "/api/v1/invitations/acceptance",
		Summary:       "Accept an invitation by choosing a password",
		Tags:          []string{"People"},
		DefaultStatus: http.StatusNoContent,
		MaxBodyBytes:  4 << 10,
	}, func(ctx context.Context, input *AcceptInviteInput) (*EmptyOutput, error) {
		if people == nil {
			return nil, huma.Error503ServiceUnavailable("Convites temporariamente indisponíveis")
		}
		if _, err := people.Accept(ctx, input.Body.Token, input.Body.Password); err != nil {
			return nil, peopleError(err)
		}
		return &EmptyOutput{Status: http.StatusNoContent}, nil
	})
}

func peopleError(err error) error {
	switch {
	case errors.Is(err, authdomain.ErrInvitationNotFound),
		errors.Is(err, authdomain.ErrAccountNotFound):
		return huma.Error404NotFound("Convite não encontrado")
	case errors.Is(err, authdomain.ErrInvitationUsed):
		return huma.Error409Conflict("Este convite já foi utilizado")
	case errors.Is(err, authdomain.ErrInvitationExpired):
		return huma.Error410Gone("Este convite expirou. Peça um novo à administração.")
	case errors.Is(err, authdomain.ErrAccountAlreadyExists):
		return huma.Error409Conflict("Este endereço já tem acesso")
	case errors.Is(err, authdomain.ErrWeakPassword):
		return huma.Error422UnprocessableEntity("A senha precisa ter ao menos 12 caracteres")
	case errors.Is(err, authdomain.ErrUnsupportedRole):
		return huma.Error422UnprocessableEntity("Papel inválido")
	case errors.Is(err, authdomain.ErrInvalidInvitation):
		return huma.Error422UnprocessableEntity("Informe nome e e-mail válidos")
	case errors.Is(err, authdomain.ErrInstallationAlreadyInitialized):
		return huma.Error409Conflict(
			"A organização precisa manter ao menos uma administração",
		)
	default:
		slog.Error("people operation failed", "error", err)
		return huma.Error503ServiceUnavailable("Pessoas temporariamente indisponíveis")
	}
}
