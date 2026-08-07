package document

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	ErrNotFound         = errors.New("document not found")
	ErrRevisionConflict = errors.New("document revision conflict")
)

type Summary struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Revision  int       `json:"revision"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Stored struct {
	ID        string    `json:"id"`
	Revision  int       `json:"revision"`
	UpdatedAt time.Time `json:"updatedAt"`
	Content   Document  `json:"content"`
}

type NewDocument struct {
	OrganizationID string
	AuthorID       string
	// TemplateRevisionID binds the document to the script revision it was
	// created from. It never changes: publishing a new revision leaves existing
	// documents exactly as they are.
	TemplateRevisionID string
	Content            Document
	ContentHash        []byte
}

type Revision struct {
	OrganizationID   string
	DocumentID       string
	AuthorID         string
	ExpectedRevision int
	Content          Document
	ContentHash      []byte
}

type Repository interface {
	List(ctx context.Context, organizationID string) ([]Summary, error)
	Create(context.Context, NewDocument) (Stored, error)
	Get(ctx context.Context, organizationID string, documentID string) (Stored, error)
	AppendRevision(context.Context, Revision) (Stored, error)
}

// Service owns the document use cases. Every method takes the organization of
// the authenticated session so a document can never be read or written across
// organizations, even when the identifier is guessed.
type Service struct {
	Repository Repository
}

func (s Service) List(ctx context.Context, organizationID string) ([]Summary, error) {
	if organizationID == "" {
		return nil, ErrNotFound
	}
	return s.Repository.List(ctx, organizationID)
}

// Create stores a document already instantiated from a script revision. The
// caller resolves the revision, so this package stays unaware of templates.
func (s Service) Create(
	ctx context.Context,
	organizationID string,
	authorID string,
	templateRevisionID string,
	content Document,
) (Stored, error) {
	if err := content.Validate(); err != nil {
		return Stored{}, err
	}
	hash, err := ContentHash(content)
	if err != nil {
		return Stored{}, err
	}
	return s.Repository.Create(ctx, NewDocument{
		OrganizationID:     organizationID,
		AuthorID:           authorID,
		TemplateRevisionID: templateRevisionID,
		Content:            content,
		ContentHash:        hash,
	})
}

func (s Service) Open(
	ctx context.Context,
	organizationID string,
	documentID string,
) (Stored, error) {
	return s.Repository.Get(ctx, organizationID, documentID)
}

// Save persists the next working revision. The caller states which revision it
// edited; a mismatch is reported as a conflict instead of overwriting work.
func (s Service) Save(
	ctx context.Context,
	organizationID string,
	authorID string,
	documentID string,
	expectedRevision int,
	content Document,
) (Stored, error) {
	if err := content.Validate(); err != nil {
		return Stored{}, err
	}
	if expectedRevision < 1 {
		return Stored{}, fmt.Errorf("%w: expected revision", ErrInvalidDocument)
	}
	hash, err := ContentHash(content)
	if err != nil {
		return Stored{}, err
	}
	return s.Repository.AppendRevision(ctx, Revision{
		OrganizationID:   organizationID,
		DocumentID:       documentID,
		AuthorID:         authorID,
		ExpectedRevision: expectedRevision,
		Content:          content,
		ContentHash:      hash,
	})
}

// ContentHash identifies a revision by content. It is stored with every
// revision so an issuance can later state exactly what was rendered.
func ContentHash(content Document) ([]byte, error) {
	encoded, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("encode document: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return sum[:], nil
}
