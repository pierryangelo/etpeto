package template

import (
	"context"
	"errors"
	"testing"
	"time"
)

type repositoryStub struct {
	draft            Draft
	getErr           error
	saved            SaveDraftCommand
	saveCalls        int
	saveErr          error
	published        PublishCommand
	publishCalls     int
	publishErr       error
	currentPublished PublishedRevision
	publishedErr     error
}

func (r *repositoryStub) List(context.Context, string) ([]Summary, error) { return nil, nil }

func (r *repositoryStub) Get(context.Context, string, string) (Draft, error) {
	return r.draft, r.getErr
}

func (r *repositoryStub) SaveDraft(_ context.Context, command SaveDraftCommand) (Draft, error) {
	r.saveCalls++
	r.saved = command
	if r.saveErr != nil {
		return Draft{}, r.saveErr
	}
	return Draft{ID: command.TemplateID, DraftVersion: command.ExpectedVersion + 1, Blueprint: command.Blueprint}, nil
}

func (r *repositoryStub) Publish(_ context.Context, command PublishCommand) (PublishedRevision, error) {
	r.publishCalls++
	r.published = command
	if r.publishErr != nil {
		return PublishedRevision{}, r.publishErr
	}
	return PublishedRevision{ID: "revision-1", TemplateID: command.TemplateID, Revision: 4}, nil
}

func (r *repositoryStub) ListPublished(context.Context, string) ([]PublishedRevision, error) {
	return nil, nil
}

func (r *repositoryStub) FindPublished(context.Context, string, string) (PublishedRevision, error) {
	return r.currentPublished, r.publishedErr
}

func (r *repositoryStub) FindPublishedForTemplate(context.Context, string, string) (PublishedRevision, error) {
	return r.currentPublished, r.publishedErr
}

func (r *repositoryStub) EnsureDefault(context.Context, string, string) error { return nil }

func draftFixture() Draft {
	return Draft{
		ID:                "template-1",
		DocumentType:      "etp",
		Name:              "Estudo Técnico Preliminar",
		DraftVersion:      3,
		DraftUpdatedAt:    time.Now(),
		Blueprint:         nestedBlueprint(),
		PublishedRevision: 3,
	}
}

func TestSaveDraftRefusesAnInvalidScriptBeforeWriting(t *testing.T) {
	repository := &repositoryStub{draft: draftFixture()}
	service := Service{Repository: repository}
	blueprint := nestedBlueprint()
	blueprint.Sections[0].Title = ""

	_, err := service.SaveDraft(context.Background(), "org-1", "template-1", "user-1", 3, blueprint)

	if !errors.Is(err, ErrInvalidBlueprint) {
		t.Fatalf("SaveDraft() error = %v, want ErrInvalidBlueprint", err)
	}
	if repository.saveCalls != 0 {
		t.Fatal("an invalid script reached the repository")
	}
}

func TestSaveDraftForwardsTheExpectedVersionForOptimisticLocking(t *testing.T) {
	repository := &repositoryStub{draft: draftFixture()}
	service := Service{Repository: repository}

	if _, err := service.SaveDraft(
		context.Background(), "org-1", "template-1", "user-1", 3, nestedBlueprint(),
	); err != nil {
		t.Fatal(err)
	}

	if repository.saved.ExpectedVersion != 3 {
		t.Fatalf("expected version = %d, want 3", repository.saved.ExpectedVersion)
	}
	if repository.saved.AuthorID != "user-1" || repository.saved.OrganizationID != "org-1" {
		t.Fatalf("save command = %#v", repository.saved)
	}
}

func TestPublishFreezesTheCurrentDraft(t *testing.T) {
	repository := &repositoryStub{
		draft:            draftFixture(),
		currentPublished: PublishedRevision{Blueprint: DefaultBlueprint()},
	}
	service := Service{Repository: repository}

	published, err := service.Publish(context.Background(), "org-1", "template-1", "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if published.Revision != 4 {
		t.Fatalf("published revision = %d", published.Revision)
	}
	// What is published is the draft, not whatever the caller happened to send.
	if len(repository.published.Blueprint.Sections) != len(nestedBlueprint().Sections) {
		t.Fatalf("published blueprint = %#v", repository.published.Blueprint)
	}
	if repository.published.AuthorID != "user-1" {
		t.Fatalf("publisher = %q", repository.published.AuthorID)
	}
}

func TestPublishRefusesWhenTheDraftMatchesWhatIsAlreadyPublished(t *testing.T) {
	draft := draftFixture()
	repository := &repositoryStub{
		draft:            draft,
		currentPublished: PublishedRevision{Blueprint: draft.Blueprint},
	}
	service := Service{Repository: repository}

	_, err := service.Publish(context.Background(), "org-1", "template-1", "user-1")

	// The revision history should record changes, not clicks.
	if !errors.Is(err, ErrNothingToPublish) {
		t.Fatalf("Publish() error = %v, want ErrNothingToPublish", err)
	}
	if repository.publishCalls != 0 {
		t.Fatal("an unchanged script was published")
	}
}

func TestPublishAcceptsTheFirstRevisionOfANewScript(t *testing.T) {
	draft := draftFixture()
	draft.PublishedRevision = 0
	repository := &repositoryStub{draft: draft}
	service := Service{Repository: repository}

	if _, err := service.Publish(context.Background(), "org-1", "template-1", "user-1"); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if repository.publishCalls != 1 {
		t.Fatal("the first revision was not published")
	}
}

func TestPublishRefusesADraftThatBecameInvalid(t *testing.T) {
	draft := draftFixture()
	draft.Blueprint.Sections[0].AllowedElements = nil
	repository := &repositoryStub{draft: draft}
	service := Service{Repository: repository}

	_, err := service.Publish(context.Background(), "org-1", "template-1", "user-1")

	if !errors.Is(err, ErrInvalidBlueprint) {
		t.Fatalf("Publish() error = %v, want ErrInvalidBlueprint", err)
	}
	if repository.publishCalls != 0 {
		t.Fatal("an invalid script was published")
	}
}
