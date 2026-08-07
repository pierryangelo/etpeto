package document

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
)

func TestServiceCreateStoresTheDefaultOutlineWithItsContentHash(t *testing.T) {
	repository := &repositoryStub{}
	service := Service{Repository: repository}

	stored, err := service.Create(context.Background(), "organization-1", "user-1", " Aquisição de notebooks ")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Revision != 1 {
		t.Fatalf("stored revision = %d", stored.Revision)
	}
	if repository.created.Content.Title != "Aquisição de notebooks" {
		t.Fatalf("created title = %q", repository.created.Content.Title)
	}
	if repository.created.OrganizationID != "organization-1" || repository.created.AuthorID != "user-1" {
		t.Fatalf("created ownership = %#v", repository.created)
	}
	expected, err := ContentHash(repository.created.Content)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(repository.created.ContentHash, expected) {
		t.Fatalf("content hash = %x, want %x", repository.created.ContentHash, expected)
	}
}

func TestServiceCreateRejectsABlankTitleBeforeWriting(t *testing.T) {
	repository := &repositoryStub{}
	service := Service{Repository: repository}

	_, err := service.Create(context.Background(), "organization-1", "user-1", "  ")
	if !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("Create() error = %v, want ErrInvalidDocument", err)
	}
	if repository.createCalls != 0 {
		t.Fatal("an invalid document was written")
	}
}

func TestServiceSaveForwardsTheExpectedRevisionForOptimisticLocking(t *testing.T) {
	repository := &repositoryStub{}
	service := Service{Repository: repository}
	content, err := New("Aquisição de notebooks")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := service.Save(context.Background(), "organization-1", "user-1", "document-1", 4, content); err != nil {
		t.Fatal(err)
	}
	if repository.appended.ExpectedRevision != 4 {
		t.Fatalf("expected revision = %d", repository.appended.ExpectedRevision)
	}
	if repository.appended.DocumentID != "document-1" || repository.appended.OrganizationID != "organization-1" {
		t.Fatalf("appended revision = %#v", repository.appended)
	}
}

func TestServiceSaveRefusesInvalidContentBeforeWriting(t *testing.T) {
	repository := &repositoryStub{}
	service := Service{Repository: repository}
	content, err := New("Aquisição de notebooks")
	if err != nil {
		t.Fatal(err)
	}
	content.Sections[0].Elements[0].Type = "planilha"

	_, err = service.Save(context.Background(), "organization-1", "user-1", "document-1", 1, content)
	if !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("Save() error = %v, want ErrInvalidDocument", err)
	}
	if repository.appendCalls != 0 {
		t.Fatal("invalid content reached the repository")
	}
}

func TestServiceSaveSurfacesRevisionConflicts(t *testing.T) {
	repository := &repositoryStub{appendErr: ErrRevisionConflict}
	service := Service{Repository: repository}
	content, err := New("Aquisição de notebooks")
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.Save(context.Background(), "organization-1", "user-1", "document-1", 1, content)
	if !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("Save() error = %v, want ErrRevisionConflict", err)
	}
}

type repositoryStub struct {
	created     NewDocument
	createCalls int
	appended    Revision
	appendCalls int
	appendErr   error
}

func (r *repositoryStub) List(context.Context, string) ([]Summary, error) {
	return nil, nil
}

func (r *repositoryStub) Create(_ context.Context, document NewDocument) (Stored, error) {
	r.createCalls++
	r.created = document
	return Stored{ID: "document-1", Revision: 1, UpdatedAt: time.Now(), Content: document.Content}, nil
}

func (r *repositoryStub) Get(context.Context, string, string) (Stored, error) {
	return Stored{}, ErrNotFound
}

func (r *repositoryStub) AppendRevision(_ context.Context, revision Revision) (Stored, error) {
	r.appendCalls++
	r.appended = revision
	if r.appendErr != nil {
		return Stored{}, r.appendErr
	}
	return Stored{
		ID:       revision.DocumentID,
		Revision: revision.ExpectedRevision + 1,
		Content:  revision.Content,
	}, nil
}
