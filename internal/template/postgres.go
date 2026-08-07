package template

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

type PostgresRepository struct {
	Pool *pgxpool.Pool
}

func (r PostgresRepository) List(
	ctx context.Context,
	organizationID string,
) ([]Summary, error) {
	parsed, err := parseUUID(organizationID)
	if err != nil {
		return nil, err
	}
	rows, err := dbgen.New(r.Pool).ListTemplates(ctx, parsed)
	if err != nil {
		return nil, err
	}
	summaries := make([]Summary, 0, len(rows))
	for _, row := range rows {
		published := 0
		if value, ok := row.PublishedRevision.(int32); ok {
			published = int(value)
		}
		summary := Summary{
			ID:                formatUUID(row.ID),
			DocumentType:      row.DocumentType,
			Name:              row.Name,
			DraftVersion:      int(row.DraftVersion),
			DraftUpdatedAt:    row.DraftUpdatedAt.Time,
			PublishedRevision: published,
			DocumentCount:     int(row.DocumentCount),
		}
		summary.HasUnpublishedChanges, err = r.draftDiffers(ctx, parsed, row.ID, published)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

// draftDiffers answers the one question the list screen needs: is the manager
// looking at changes the authors have not received yet?
func (r PostgresRepository) draftDiffers(
	ctx context.Context,
	organizationID pgtype.UUID,
	templateID pgtype.UUID,
	publishedRevision int,
) (bool, error) {
	if publishedRevision == 0 {
		return true, nil
	}
	draft, err := r.Get(ctx, formatUUID(organizationID), formatUUID(templateID))
	if err != nil {
		return false, err
	}
	published, err := r.FindPublishedForTemplate(
		ctx,
		formatUUID(organizationID),
		formatUUID(templateID),
	)
	if errors.Is(err, ErrNotFound) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return !sameBlueprint(published.Blueprint, draft.Blueprint), nil
}

func (r PostgresRepository) Get(
	ctx context.Context,
	organizationID string,
	templateID string,
) (Draft, error) {
	parsedOrganization, err := parseUUID(organizationID)
	if err != nil {
		return Draft{}, err
	}
	parsedTemplate, err := parseUUID(templateID)
	if err != nil {
		return Draft{}, ErrNotFound
	}
	row, err := dbgen.New(r.Pool).FindTemplate(ctx, dbgen.FindTemplateParams{
		ID:             parsedTemplate,
		OrganizationID: parsedOrganization,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Draft{}, ErrNotFound
	}
	if err != nil {
		return Draft{}, err
	}
	var blueprint Blueprint
	if err := json.Unmarshal(row.Draft, &blueprint); err != nil {
		return Draft{}, fmt.Errorf("decode stored draft: %w", err)
	}
	draft := Draft{
		ID:             formatUUID(row.ID),
		DocumentType:   row.DocumentType,
		Name:           row.Name,
		DraftVersion:   int(row.DraftVersion),
		DraftUpdatedAt: row.DraftUpdatedAt.Time,
		Blueprint:      blueprint,
	}
	published, err := r.FindPublishedForTemplate(ctx, organizationID, templateID)
	if err == nil {
		draft.PublishedRevision = published.Revision
	} else if !errors.Is(err, ErrNotFound) {
		return Draft{}, err
	}
	return draft, nil
}

func (r PostgresRepository) SaveDraft(
	ctx context.Context,
	command SaveDraftCommand,
) (Draft, error) {
	organizationID, err := parseUUID(command.OrganizationID)
	if err != nil {
		return Draft{}, err
	}
	templateID, err := parseUUID(command.TemplateID)
	if err != nil {
		return Draft{}, ErrNotFound
	}
	authorID, err := parseUUID(command.AuthorID)
	if err != nil {
		return Draft{}, err
	}
	encoded, err := json.Marshal(command.Blueprint)
	if err != nil {
		return Draft{}, fmt.Errorf("encode draft: %w", err)
	}

	row, err := dbgen.New(r.Pool).SaveTemplateDraft(ctx, dbgen.SaveTemplateDraftParams{
		ID:             templateID,
		OrganizationID: organizationID,
		Draft:          encoded,
		DraftUpdatedBy: authorID,
		DraftVersion:   int32(command.ExpectedVersion),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// No row means either the script is not here or someone else saved a
		// newer draft. The two are told apart, never merged into one message.
		exists, existsErr := dbgen.New(r.Pool).TemplateExists(ctx, dbgen.TemplateExistsParams{
			ID:             templateID,
			OrganizationID: organizationID,
		})
		if existsErr != nil {
			return Draft{}, existsErr
		}
		if !exists {
			return Draft{}, ErrNotFound
		}
		return Draft{}, ErrConflict
	}
	if err != nil {
		return Draft{}, err
	}
	return Draft{
		ID:             formatUUID(row.ID),
		DocumentType:   row.DocumentType,
		Name:           row.Name,
		DraftVersion:   int(row.DraftVersion),
		DraftUpdatedAt: row.DraftUpdatedAt.Time,
		Blueprint:      command.Blueprint,
	}, nil
}

func (r PostgresRepository) Publish(
	ctx context.Context,
	command PublishCommand,
) (PublishedRevision, error) {
	templateID, err := parseUUID(command.TemplateID)
	if err != nil {
		return PublishedRevision{}, ErrNotFound
	}
	authorID, err := parseUUID(command.AuthorID)
	if err != nil {
		return PublishedRevision{}, err
	}
	encoded, err := json.Marshal(command.Blueprint)
	if err != nil {
		return PublishedRevision{}, fmt.Errorf("encode blueprint: %w", err)
	}

	row, err := dbgen.New(r.Pool).PublishTemplateRevision(ctx, dbgen.PublishTemplateRevisionParams{
		TemplateID:    templateID,
		Blueprint:     encoded,
		SchemaVersion: int32(command.SchemaVersion),
		PublishedBy:   authorID,
	})
	if err != nil {
		return PublishedRevision{}, err
	}
	return PublishedRevision{
		ID:          formatUUID(row.ID),
		TemplateID:  formatUUID(row.TemplateID),
		Revision:    int(row.Revision),
		PublishedAt: row.PublishedAt.Time,
		Blueprint:   command.Blueprint,
	}, nil
}

func (r PostgresRepository) ListPublished(
	ctx context.Context,
	organizationID string,
) ([]PublishedRevision, error) {
	parsed, err := parseUUID(organizationID)
	if err != nil {
		return nil, err
	}
	rows, err := dbgen.New(r.Pool).ListPublishedTemplates(ctx, parsed)
	if err != nil {
		return nil, err
	}
	revisions := make([]PublishedRevision, 0, len(rows))
	for _, row := range rows {
		revisions = append(revisions, PublishedRevision{
			ID:           formatUUID(row.RevisionID),
			TemplateID:   formatUUID(row.TemplateID),
			TemplateName: row.Name,
			DocumentType: row.DocumentType,
			Revision:     int(row.Revision),
			PublishedAt:  row.PublishedAt.Time,
		})
	}
	return revisions, nil
}

func (r PostgresRepository) FindPublished(
	ctx context.Context,
	organizationID string,
	revisionID string,
) (PublishedRevision, error) {
	parsedOrganization, err := parseUUID(organizationID)
	if err != nil {
		return PublishedRevision{}, err
	}
	parsedRevision, err := parseUUID(revisionID)
	if err != nil {
		return PublishedRevision{}, ErrNotFound
	}
	row, err := dbgen.New(r.Pool).FindPublishedRevision(ctx, dbgen.FindPublishedRevisionParams{
		ID:             parsedRevision,
		OrganizationID: parsedOrganization,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return PublishedRevision{}, ErrNotFound
	}
	if err != nil {
		return PublishedRevision{}, err
	}
	return decodeRevision(row.ID, row.TemplateID, row.Revision, row.PublishedAt, row.Blueprint)
}

func (r PostgresRepository) FindPublishedForTemplate(
	ctx context.Context,
	organizationID string,
	templateID string,
) (PublishedRevision, error) {
	draft, err := r.rawTemplate(ctx, organizationID, templateID)
	if err != nil {
		return PublishedRevision{}, err
	}
	parsedOrganization, err := parseUUID(organizationID)
	if err != nil {
		return PublishedRevision{}, err
	}
	row, err := dbgen.New(r.Pool).FindLatestPublishedRevision(
		ctx,
		dbgen.FindLatestPublishedRevisionParams{
			OrganizationID: parsedOrganization,
			DocumentType:   draft.DocumentType,
		},
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublishedRevision{}, ErrNotFound
	}
	if err != nil {
		return PublishedRevision{}, err
	}
	return decodeRevision(row.ID, row.TemplateID, row.Revision, row.PublishedAt, row.Blueprint)
}

func (r PostgresRepository) rawTemplate(
	ctx context.Context,
	organizationID string,
	templateID string,
) (dbgen.FindTemplateRow, error) {
	parsedOrganization, err := parseUUID(organizationID)
	if err != nil {
		return dbgen.FindTemplateRow{}, err
	}
	parsedTemplate, err := parseUUID(templateID)
	if err != nil {
		return dbgen.FindTemplateRow{}, ErrNotFound
	}
	row, err := dbgen.New(r.Pool).FindTemplate(ctx, dbgen.FindTemplateParams{
		ID:             parsedTemplate,
		OrganizationID: parsedOrganization,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return dbgen.FindTemplateRow{}, ErrNotFound
	}
	return row, err
}

// EnsureDefault gives a new installation a published ETP script, so the first
// author can create a document before anyone opens the management screen. It is
// idempotent: an organization that already has the script is left alone.
func (r PostgresRepository) EnsureDefault(
	ctx context.Context,
	organizationID string,
	authorID string,
) error {
	parsedOrganization, err := parseUUID(organizationID)
	if err != nil {
		return err
	}
	parsedAuthor, err := parseUUID(authorID)
	if err != nil {
		return err
	}
	queries := dbgen.New(r.Pool)
	if _, err := queries.FindTemplateByType(ctx, dbgen.FindTemplateByTypeParams{
		OrganizationID: parsedOrganization,
		DocumentType:   "etp",
	}); err == nil {
		return nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	blueprint := DefaultBlueprint()
	encoded, err := json.Marshal(blueprint)
	if err != nil {
		return err
	}
	transaction, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	transactional := queries.WithTx(transaction)
	created, err := transactional.CreateTemplate(ctx, dbgen.CreateTemplateParams{
		OrganizationID: parsedOrganization,
		DocumentType:   blueprint.DocumentType,
		Name:           "Estudo Técnico Preliminar",
		Draft:          encoded,
		DraftUpdatedBy: parsedAuthor,
	})
	if err != nil {
		return err
	}
	if _, err := transactional.PublishTemplateRevision(ctx, dbgen.PublishTemplateRevisionParams{
		TemplateID:    created.ID,
		Blueprint:     encoded,
		SchemaVersion: int32(blueprint.SchemaVersion),
		PublishedBy:   parsedAuthor,
	}); err != nil {
		return err
	}
	return transaction.Commit(ctx)
}

func decodeRevision(
	id pgtype.UUID,
	templateID pgtype.UUID,
	revision int32,
	publishedAt pgtype.Timestamptz,
	encoded []byte,
) (PublishedRevision, error) {
	var blueprint Blueprint
	if err := json.Unmarshal(encoded, &blueprint); err != nil {
		return PublishedRevision{}, fmt.Errorf("decode published blueprint: %w", err)
	}
	return PublishedRevision{
		ID:          formatUUID(id),
		TemplateID:  formatUUID(templateID),
		Revision:    int(revision),
		PublishedAt: publishedAt.Time,
		Blueprint:   blueprint,
	}, nil
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
