// Package document holds the canonical Etpeto document model. The model is
// independent from Lexical and from Typst: Lexical state only ever lives inside
// a rich text element, and Typst is a rendering target reached through
// normalization, never a source of truth.
package document

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	// SchemaVersion is the only document schema the application writes. Reading
	// an older supported version stays a migration decision, never an implicit
	// rewrite.
	SchemaVersion = 1

	DocumentTypeETP = "etp"

	ElementRichText = "richText"
)

const (
	maxTitleRunes       = 300
	maxSectionTitle     = 300
	maxSections         = 100
	maxElementsPerBlock = 50
	maxRichTextBytes    = 200_000
)

var ErrInvalidDocument = errors.New("invalid document")

type Document struct {
	SchemaVersion int       `json:"schemaVersion"`
	DocumentType  string    `json:"documentType"`
	Title         string    `json:"title"`
	Sections      []Section `json:"sections"`
}

type Section struct {
	ID            string    `json:"id"`
	DefinitionKey string    `json:"definitionKey"`
	Title         string    `json:"title"`
	Guidance      string    `json:"guidance,omitempty"`
	Elements      []Element `json:"elements"`
}

// Element is a discriminated variant. Only rich text exists today; price
// research, risk maps and tables enter as further variants with their own
// validation, never as free-form JSON.
type Element struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	RichText json.RawMessage `json:"richText,omitempty"`
}

type outlineSection struct {
	definitionKey string
	title         string
	guidance      string
}

// defaultOutline is the provisional ETP script. Configurable, versioned scripts
// per organization replace it later; until then every document is created from
// this list so the model already carries stable definition keys.
var defaultOutline = []outlineSection{
	{
		definitionKey: "necessidade",
		title:         "Necessidade da contratação",
		guidance:      "Descreva o problema a resolver e o resultado esperado para a administração.",
	},
	{
		definitionKey: "requisitos",
		title:         "Requisitos da contratação",
		guidance:      "Registre requisitos funcionais, técnicos e de sustentabilidade aplicáveis.",
	},
	{
		definitionKey: "pesquisa-de-precos",
		title:         "Pesquisa de preços",
		guidance:      "Descreva fontes consultadas, datas, valores observados e o critério de cálculo.",
	},
	{
		definitionKey: "mapa-de-riscos",
		title:         "Mapa de riscos",
		guidance:      "Relacione riscos, probabilidade, impacto, ações preventivas e de contingência.",
	},
	{
		definitionKey: "conclusao",
		title:         "Conclusão",
		guidance:      "Registre a conclusão sobre a viabilidade da contratação.",
	},
}

func New(title string) (Document, error) {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" || utf8.RuneCountInString(trimmed) > maxTitleRunes {
		return Document{}, fmt.Errorf("%w: title", ErrInvalidDocument)
	}
	sections := make([]Section, 0, len(defaultOutline))
	for _, outline := range defaultOutline {
		sections = append(sections, Section{
			ID:            uuid.NewString(),
			DefinitionKey: outline.definitionKey,
			Title:         outline.title,
			Guidance:      outline.guidance,
			Elements: []Element{{
				ID:   uuid.NewString(),
				Type: ElementRichText,
			}},
		})
	}
	return Document{
		SchemaVersion: SchemaVersion,
		DocumentType:  DocumentTypeETP,
		Title:         trimmed,
		Sections:      sections,
	}, nil
}

func (d Document) Validate() error {
	if d.SchemaVersion != SchemaVersion {
		return fmt.Errorf("%w: unsupported schema version", ErrInvalidDocument)
	}
	if d.DocumentType != DocumentTypeETP {
		return fmt.Errorf("%w: unsupported document type", ErrInvalidDocument)
	}
	title := strings.TrimSpace(d.Title)
	if title == "" || utf8.RuneCountInString(title) > maxTitleRunes {
		return fmt.Errorf("%w: title", ErrInvalidDocument)
	}
	if len(d.Sections) == 0 || len(d.Sections) > maxSections {
		return fmt.Errorf("%w: section count", ErrInvalidDocument)
	}
	seen := make(map[string]bool, len(d.Sections))
	for index, section := range d.Sections {
		if err := section.validate(seen); err != nil {
			return fmt.Errorf("%w in section %d", err, index)
		}
	}
	return nil
}

func (s Section) validate(seen map[string]bool) error {
	if s.ID == "" || seen[s.ID] {
		return fmt.Errorf("%w: section identifier", ErrInvalidDocument)
	}
	seen[s.ID] = true
	if strings.TrimSpace(s.DefinitionKey) == "" {
		return fmt.Errorf("%w: section definition key", ErrInvalidDocument)
	}
	title := strings.TrimSpace(s.Title)
	if title == "" || utf8.RuneCountInString(title) > maxSectionTitle {
		return fmt.Errorf("%w: section title", ErrInvalidDocument)
	}
	if len(s.Elements) > maxElementsPerBlock {
		return fmt.Errorf("%w: element count", ErrInvalidDocument)
	}
	for _, element := range s.Elements {
		if err := element.validate(seen); err != nil {
			return err
		}
	}
	return nil
}

func (e Element) validate(seen map[string]bool) error {
	if e.ID == "" || seen[e.ID] {
		return fmt.Errorf("%w: element identifier", ErrInvalidDocument)
	}
	seen[e.ID] = true
	if e.Type != ElementRichText {
		return fmt.Errorf("%w: unsupported element type", ErrInvalidDocument)
	}
	if len(e.RichText) == 0 {
		return nil
	}
	if len(e.RichText) > maxRichTextBytes {
		return fmt.Errorf("%w: rich text size", ErrInvalidDocument)
	}
	// The editor state is stored opaquely, but it must still be well-formed
	// JSON with a root: the API is the boundary that refuses malformed content.
	var envelope struct {
		Root json.RawMessage `json:"root"`
	}
	if err := json.Unmarshal(e.RichText, &envelope); err != nil || len(envelope.Root) == 0 {
		return fmt.Errorf("%w: rich text state", ErrInvalidDocument)
	}
	return nil
}
