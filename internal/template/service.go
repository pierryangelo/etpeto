package template

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Summary is what the management screen lists: the state of the script, not its
// content.
type Summary struct {
	ID                string    `json:"id"`
	DocumentType      string    `json:"documentType"`
	Name              string    `json:"name"`
	DraftVersion      int       `json:"draftVersion"`
	DraftUpdatedAt    time.Time `json:"draftUpdatedAt"`
	PublishedRevision int       `json:"publishedRevision"`
	DocumentCount     int       `json:"documentCount"`
	// HasUnpublishedChanges tells the manager whether the draft still differs
	// from what the authors are using.
	HasUnpublishedChanges bool `json:"hasUnpublishedChanges"`
}

type Draft struct {
	ID             string    `json:"id"`
	DocumentType   string    `json:"documentType"`
	Name           string    `json:"name"`
	DraftVersion   int       `json:"draftVersion"`
	DraftUpdatedAt time.Time `json:"draftUpdatedAt"`
	Blueprint      Blueprint `json:"blueprint"`
	// PublishedRevision is 0 while the script has never been published.
	PublishedRevision int `json:"publishedRevision"`
}

type PublishedRevision struct {
	ID           string    `json:"id"`
	TemplateID   string    `json:"templateId"`
	TemplateName string    `json:"templateName"`
	DocumentType string    `json:"documentType"`
	Revision     int       `json:"revision"`
	PublishedAt  time.Time `json:"publishedAt"`
	Blueprint    Blueprint `json:"-"`
}

type Repository interface {
	List(ctx context.Context, organizationID string) ([]Summary, error)
	Get(ctx context.Context, organizationID string, templateID string) (Draft, error)
	SaveDraft(ctx context.Context, command SaveDraftCommand) (Draft, error)
	Publish(ctx context.Context, command PublishCommand) (PublishedRevision, error)
	ListPublished(ctx context.Context, organizationID string) ([]PublishedRevision, error)
	FindPublished(ctx context.Context, organizationID string, revisionID string) (PublishedRevision, error)
	FindPublishedForTemplate(ctx context.Context, organizationID string, templateID string) (PublishedRevision, error)
	EnsureDefault(ctx context.Context, organizationID string, authorID string) error
}

type SaveDraftCommand struct {
	OrganizationID  string
	TemplateID      string
	AuthorID        string
	ExpectedVersion int
	Blueprint       Blueprint
}

type PublishCommand struct {
	OrganizationID string
	TemplateID     string
	AuthorID       string
	Blueprint      Blueprint
	SchemaVersion  int
}

// Service owns the script use cases. Editing a draft never affects an author:
// only publishing creates the immutable revision they will use.
type Service struct {
	Repository Repository
}

func (s Service) List(ctx context.Context, organizationID string) ([]Summary, error) {
	return s.Repository.List(ctx, organizationID)
}

func (s Service) Get(
	ctx context.Context,
	organizationID string,
	templateID string,
) (Draft, error) {
	return s.Repository.Get(ctx, organizationID, templateID)
}

func (s Service) SaveDraft(
	ctx context.Context,
	organizationID string,
	templateID string,
	authorID string,
	expectedVersion int,
	blueprint Blueprint,
) (Draft, error) {
	if err := blueprint.Validate(); err != nil {
		return Draft{}, err
	}
	if expectedVersion < 1 {
		return Draft{}, fmt.Errorf("%w: expected draft version", ErrInvalidBlueprint)
	}
	return s.Repository.SaveDraft(ctx, SaveDraftCommand{
		OrganizationID:  organizationID,
		TemplateID:      templateID,
		AuthorID:        authorID,
		ExpectedVersion: expectedVersion,
		Blueprint:       blueprint,
	})
}

// Publish freezes the current draft as the next revision. It refuses when the
// draft is identical to what is already published, so the revision history
// records changes rather than clicks.
func (s Service) Publish(
	ctx context.Context,
	organizationID string,
	templateID string,
	authorID string,
) (PublishedRevision, error) {
	draft, err := s.Repository.Get(ctx, organizationID, templateID)
	if err != nil {
		return PublishedRevision{}, err
	}
	if err := draft.Blueprint.Validate(); err != nil {
		return PublishedRevision{}, err
	}
	if draft.PublishedRevision > 0 {
		published, err := s.Repository.FindPublishedForTemplate(ctx, organizationID, templateID)
		if err == nil && sameBlueprint(published.Blueprint, draft.Blueprint) {
			return PublishedRevision{}, ErrNothingToPublish
		}
	}
	return s.Repository.Publish(ctx, PublishCommand{
		OrganizationID: organizationID,
		TemplateID:     templateID,
		AuthorID:       authorID,
		Blueprint:      draft.Blueprint,
		SchemaVersion:  draft.Blueprint.SchemaVersion,
	})
}

func (s Service) ListPublished(
	ctx context.Context,
	organizationID string,
) ([]PublishedRevision, error) {
	return s.Repository.ListPublished(ctx, organizationID)
}

func (s Service) FindPublished(
	ctx context.Context,
	organizationID string,
	revisionID string,
) (PublishedRevision, error) {
	return s.Repository.FindPublished(ctx, organizationID, revisionID)
}

// sameBlueprint compares by encoded content. Field order is stable because both
// sides are marshalled from the same Go type.
func sameBlueprint(left Blueprint, right Blueprint) bool {
	encodedLeft, leftErr := json.Marshal(left)
	encodedRight, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(encodedLeft) == string(encodedRight)
}
