package document

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/etpeto/etpeto/internal/platform/database/dbgen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository stores documents and their revisions. Appending a revision
// spans two statements, so it needs the pool to open a transaction.
type PostgresRepository struct {
	Pool *pgxpool.Pool
}

func (r PostgresRepository) List(ctx context.Context, organizationID string) ([]Summary, error) {
	parsedOrganizationID, err := parseUUID(organizationID)
	if err != nil {
		return nil, err
	}
	rows, err := dbgen.New(r.Pool).ListDocuments(ctx, parsedOrganizationID)
	if err != nil {
		return nil, err
	}
	summaries := make([]Summary, 0, len(rows))
	for _, row := range rows {
		summaries = append(summaries, Summary{
			ID:        formatUUID(row.ID),
			Title:     row.Title,
			Revision:  int(row.CurrentRevision),
			UpdatedAt: row.UpdatedAt.Time,
		})
	}
	return summaries, nil
}

func (r PostgresRepository) Create(ctx context.Context, document NewDocument) (Stored, error) {
	organizationID, err := parseUUID(document.OrganizationID)
	if err != nil {
		return Stored{}, err
	}
	authorID, err := parseUUID(document.AuthorID)
	if err != nil {
		return Stored{}, err
	}
	content, err := json.Marshal(document.Content)
	if err != nil {
		return Stored{}, fmt.Errorf("encode document: %w", err)
	}

	transaction, err := r.Pool.Begin(ctx)
	if err != nil {
		return Stored{}, err
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	queries := dbgen.New(r.Pool).WithTx(transaction)
	created, err := queries.CreateDocument(ctx, dbgen.CreateDocumentParams{
		OrganizationID: organizationID,
		CreatedBy:      authorID,
		DocumentType:   document.Content.DocumentType,
		Title:          document.Content.Title,
	})
	if err != nil {
		return Stored{}, err
	}
	if _, err := queries.CreateDocumentRevision(ctx, dbgen.CreateDocumentRevisionParams{
		DocumentID:    created.ID,
		Revision:      created.CurrentRevision,
		SchemaVersion: int32(document.Content.SchemaVersion),
		Content:       content,
		ContentHash:   document.ContentHash,
		CreatedBy:     authorID,
	}); err != nil {
		return Stored{}, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return Stored{}, err
	}
	return Stored{
		ID:        formatUUID(created.ID),
		Revision:  int(created.CurrentRevision),
		UpdatedAt: created.UpdatedAt.Time,
		Content:   document.Content,
	}, nil
}

func (r PostgresRepository) Get(
	ctx context.Context,
	organizationID string,
	documentID string,
) (Stored, error) {
	parsedOrganizationID, err := parseUUID(organizationID)
	if err != nil {
		return Stored{}, err
	}
	parsedDocumentID, err := parseUUID(documentID)
	if err != nil {
		return Stored{}, ErrNotFound
	}
	row, err := dbgen.New(r.Pool).FindDocument(ctx, dbgen.FindDocumentParams{
		ID:             parsedDocumentID,
		OrganizationID: parsedOrganizationID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Stored{}, ErrNotFound
	}
	if err != nil {
		return Stored{}, err
	}
	var content Document
	if err := json.Unmarshal(row.Content, &content); err != nil {
		return Stored{}, fmt.Errorf("decode stored document: %w", err)
	}
	return Stored{
		ID:        formatUUID(row.ID),
		Revision:  int(row.CurrentRevision),
		UpdatedAt: row.UpdatedAt.Time,
		Content:   content,
	}, nil
}

// AppendRevision advances the document only when its current revision still
// matches the one the caller edited. A mismatch is a conflict, never a silent
// overwrite of someone else's work.
func (r PostgresRepository) AppendRevision(
	ctx context.Context,
	revision Revision,
) (Stored, error) {
	organizationID, err := parseUUID(revision.OrganizationID)
	if err != nil {
		return Stored{}, err
	}
	documentID, err := parseUUID(revision.DocumentID)
	if err != nil {
		return Stored{}, ErrNotFound
	}
	authorID, err := parseUUID(revision.AuthorID)
	if err != nil {
		return Stored{}, err
	}
	content, err := json.Marshal(revision.Content)
	if err != nil {
		return Stored{}, fmt.Errorf("encode document: %w", err)
	}

	transaction, err := r.Pool.Begin(ctx)
	if err != nil {
		return Stored{}, err
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	queries := dbgen.New(r.Pool).WithTx(transaction)
	advanced, err := queries.AdvanceDocumentRevision(ctx, dbgen.AdvanceDocumentRevisionParams{
		ID:              documentID,
		OrganizationID:  organizationID,
		Title:           revision.Content.Title,
		CurrentRevision: int32(revision.ExpectedRevision),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Stored{}, r.classifyMissingUpdate(ctx, documentID, organizationID)
	}
	if err != nil {
		return Stored{}, err
	}
	if _, err := queries.CreateDocumentRevision(ctx, dbgen.CreateDocumentRevisionParams{
		DocumentID:    advanced.ID,
		Revision:      advanced.CurrentRevision,
		SchemaVersion: int32(revision.Content.SchemaVersion),
		Content:       content,
		ContentHash:   revision.ContentHash,
		CreatedBy:     authorID,
	}); err != nil {
		return Stored{}, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return Stored{}, err
	}
	return Stored{
		ID:        formatUUID(advanced.ID),
		Revision:  int(advanced.CurrentRevision),
		UpdatedAt: advanced.UpdatedAt.Time,
		Content:   revision.Content,
	}, nil
}

// classifyMissingUpdate separates "there is no such document in this
// organization" from "someone else already saved a newer revision".
func (r PostgresRepository) classifyMissingUpdate(
	ctx context.Context,
	documentID pgtype.UUID,
	organizationID pgtype.UUID,
) error {
	exists, err := dbgen.New(r.Pool).DocumentExists(ctx, dbgen.DocumentExistsParams{
		ID:             documentID,
		OrganizationID: organizationID,
	})
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return ErrRevisionConflict
}

func formatUUID(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	return uuid.UUID(value.Bytes).String()
}

func parseUUID(value string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("parse identifier: %w", err)
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}
