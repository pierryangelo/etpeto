package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestProvisionCreatesOrganizationAdministratorFromAnEmptyInstallation(t *testing.T) {
	repository := &provisioningStub{organizationErr: ErrOrganizationNotFound, accountErr: ErrAccountNotFound}
	provisioner := Provisioner{Repository: repository, Passwords: testPasswordHasher()}

	outcome, err := provisioner.Provision(context.Background(), ProvisionRequest{
		OrganizationSlug: "orgao-exemplo",
		OrganizationName: "Órgão de Exemplo",
		Email:            "  ADMIN@ORGAO.GOV.BR ",
		DisplayName:      "Pessoa Administradora",
		Password:         "senha institucional",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.Created {
		t.Fatal("outcome did not report a created administrator")
	}
	if repository.createdOrganization.Slug != "orgao-exemplo" {
		t.Fatalf("created organization = %#v", repository.createdOrganization)
	}
	created := repository.createdAdministrator
	if created.EmailNormalized != "admin@orgao.gov.br" || created.Email != "ADMIN@ORGAO.GOV.BR" {
		t.Fatalf("created administrator email = %#v", created)
	}
	if created.Role != RoleOrganizationAdmin {
		t.Fatalf("created role = %q", created.Role)
	}
	if strings.Contains(created.PasswordHash, "senha institucional") {
		t.Fatal("raw password was stored")
	}
	verified, err := testPasswordHasher().Verify(created.PasswordHash, "senha institucional")
	if err != nil || !verified {
		t.Fatalf("stored hash did not verify the provisioned password: %v", err)
	}
}

func TestProvisionIsIdempotentForTheSameAdministrator(t *testing.T) {
	repository := &provisioningStub{organizationID: "organization-1", accountUserID: "user-1"}
	provisioner := Provisioner{Repository: repository, Passwords: testPasswordHasher()}

	outcome, err := provisioner.Provision(context.Background(), validProvisionRequest())
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Created {
		t.Fatal("an existing administrator was reported as created")
	}
	if repository.createCalls != 0 {
		t.Fatal("an existing administrator was written again")
	}
	if outcome.UserID != "user-1" {
		t.Fatalf("outcome = %#v", outcome)
	}
}

func TestProvisionRefusesASecondIncompatibleInitialization(t *testing.T) {
	repository := &provisioningStub{
		organizationID: "organization-1",
		accountErr:     ErrAccountNotFound,
		administrators: 1,
	}
	provisioner := Provisioner{Repository: repository, Passwords: testPasswordHasher()}

	_, err := provisioner.Provision(context.Background(), validProvisionRequest())
	if !errors.Is(err, ErrInstallationAlreadyInitialized) {
		t.Fatalf("Provision() error = %v, want ErrInstallationAlreadyInitialized", err)
	}
	if repository.createCalls != 0 {
		t.Fatal("a second administrator was written")
	}
}

func TestProvisionValidatesRequestBeforeTouchingTheDatabase(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*ProvisionRequest)
		wantErr error
	}{
		{
			name:    "short password",
			mutate:  func(r *ProvisionRequest) { r.Password = "curta" },
			wantErr: ErrWeakPassword,
		},
		{
			name:    "blank email",
			mutate:  func(r *ProvisionRequest) { r.Email = "   " },
			wantErr: ErrInvalidProvisionRequest,
		},
		{
			name:    "blank display name",
			mutate:  func(r *ProvisionRequest) { r.DisplayName = "" },
			wantErr: ErrInvalidProvisionRequest,
		},
		{
			name:    "blank organization slug",
			mutate:  func(r *ProvisionRequest) { r.OrganizationSlug = "" },
			wantErr: ErrInvalidProvisionRequest,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &provisioningStub{organizationErr: ErrOrganizationNotFound, accountErr: ErrAccountNotFound}
			provisioner := Provisioner{Repository: repository, Passwords: testPasswordHasher()}
			request := validProvisionRequest()
			test.mutate(&request)

			_, err := provisioner.Provision(context.Background(), request)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Provision() error = %v, want %v", err, test.wantErr)
			}
			if repository.lookupCalls != 0 || repository.createCalls != 0 {
				t.Fatal("an invalid request reached the database")
			}
		})
	}
}

func validProvisionRequest() ProvisionRequest {
	return ProvisionRequest{
		OrganizationSlug: "orgao-exemplo",
		OrganizationName: "Órgão de Exemplo",
		Email:            "admin@orgao.gov.br",
		DisplayName:      "Pessoa Administradora",
		Password:         "senha institucional",
	}
}

type provisioningStub struct {
	organizationID       string
	organizationErr      error
	accountUserID        string
	accountErr           error
	administrators       int
	lookupCalls          int
	createCalls          int
	createdOrganization  NewOrganization
	createdAdministrator NewAdministrator
}

func (s *provisioningStub) FindOrganizationBySlug(_ context.Context, _ string) (string, error) {
	s.lookupCalls++
	return s.organizationID, s.organizationErr
}

func (s *provisioningStub) CreateOrganization(_ context.Context, organization NewOrganization) (string, error) {
	s.createdOrganization = organization
	return "organization-1", nil
}

func (s *provisioningStub) FindLocalUserID(_ context.Context, _ string, _ string) (string, error) {
	s.lookupCalls++
	return s.accountUserID, s.accountErr
}

func (s *provisioningStub) CountAdministrators(_ context.Context, _ string) (int, error) {
	s.lookupCalls++
	return s.administrators, nil
}

func (s *provisioningStub) CreateAdministrator(_ context.Context, administrator NewAdministrator) (string, error) {
	s.createCalls++
	s.createdAdministrator = administrator
	return "user-1", nil
}
