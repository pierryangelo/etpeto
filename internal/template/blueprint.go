// Package template holds the institutional script: which sections a document
// has, in what order, nested how, with what guidance and which of them are
// required. A published revision is immutable, and a document stays bound to
// the revision it was created from.
package template

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/etpeto/etpeto/internal/document"
	"github.com/google/uuid"
)

const (
	// BlueprintSchemaVersion versions the stored script itself, separately from
	// the document schema: a script can gain fields without rewriting documents.
	BlueprintSchemaVersion = 1

	maxTemplateNameRunes = 200
	maxSectionKeyRunes   = 120
	maxTitleRunes        = 300
	maxGuidanceRunes     = 2000
	maxSections          = 100
	maxDepth             = 4
)

var (
	ErrInvalidBlueprint = errors.New("invalid blueprint")
	ErrNotFound         = errors.New("template not found")
	ErrNothingToPublish = errors.New("draft matches the published revision")
	ErrConflict         = errors.New("template draft changed elsewhere")
)

// allowedElementTypes are the element kinds a section may accept. Rich text is
// the only one implemented; the others are declared so a script can already be
// written for them and refuse anything unknown.
var allowedElementTypes = map[string]bool{
	document.ElementRichText: true,
	"priceResearch":          true,
	"riskMap":                true,
	"table":                  true,
}

// Section is the script's unit. A subsection is a section nested inside another
// one, exactly as in the document: same shape, same rules.
type Section struct {
	// Key is the stable meaning of the section. It survives renaming and is
	// what a document carries as its definitionKey.
	Key      string `json:"key"`
	Title    string `json:"title"`
	Guidance string `json:"guidance,omitempty"`
	// Required blocks finishing the document while the section is empty. It
	// says nothing about the subsections, which declare their own obligation.
	Required bool `json:"required"`
	// AllowRename lets the author change the visible title without changing
	// what the section means.
	AllowRename     bool      `json:"allowRename"`
	AllowedElements []string  `json:"allowedElements"`
	Sections        []Section `json:"sections,omitempty"`
}

type Blueprint struct {
	SchemaVersion int       `json:"schemaVersion"`
	DocumentType  string    `json:"documentType"`
	Sections      []Section `json:"sections"`
}

func (b Blueprint) Validate() error {
	if b.SchemaVersion != BlueprintSchemaVersion {
		return fmt.Errorf("%w: unsupported schema version", ErrInvalidBlueprint)
	}
	if b.DocumentType != document.DocumentTypeETP {
		return fmt.Errorf("%w: unsupported document type", ErrInvalidBlueprint)
	}
	if len(b.Sections) == 0 {
		return fmt.Errorf("%w: a script needs at least one section", ErrInvalidBlueprint)
	}
	if b.count() > maxSections {
		return fmt.Errorf("%w: section count", ErrInvalidBlueprint)
	}
	keys := make(map[string]bool)
	for index, section := range b.Sections {
		if err := section.validate(keys, 1); err != nil {
			return fmt.Errorf("%w in section %d", err, index)
		}
	}
	return nil
}

func (s Section) validate(keys map[string]bool, depth int) error {
	if depth > maxDepth {
		return fmt.Errorf("%w: outline depth", ErrInvalidBlueprint)
	}
	key := strings.TrimSpace(s.Key)
	// The key identifies the meaning across revisions, so it must be unique in
	// the whole script and not only among siblings.
	if key == "" || utf8.RuneCountInString(key) > maxSectionKeyRunes || keys[key] {
		return fmt.Errorf("%w: section key", ErrInvalidBlueprint)
	}
	keys[key] = true

	title := strings.TrimSpace(s.Title)
	if title == "" || utf8.RuneCountInString(title) > maxTitleRunes {
		return fmt.Errorf("%w: section title", ErrInvalidBlueprint)
	}
	if utf8.RuneCountInString(s.Guidance) > maxGuidanceRunes {
		return fmt.Errorf("%w: section guidance", ErrInvalidBlueprint)
	}
	if len(s.AllowedElements) == 0 {
		return fmt.Errorf("%w: section allows no content", ErrInvalidBlueprint)
	}
	for _, element := range s.AllowedElements {
		if !allowedElementTypes[element] {
			return fmt.Errorf("%w: unsupported element type %q", ErrInvalidBlueprint, element)
		}
	}
	for index, child := range s.Sections {
		if err := child.validate(keys, depth+1); err != nil {
			return fmt.Errorf("%w in subsection %d", err, index)
		}
	}
	return nil
}

func (b Blueprint) count() int {
	total := 0
	var walk func(sections []Section)
	walk = func(sections []Section) {
		for _, section := range sections {
			total++
			walk(section.Sections)
		}
	}
	walk(b.Sections)
	return total
}

// Instantiate turns the script into an empty document. Identifiers are minted
// per document, so two documents from the same revision never share one; the
// section key is what ties them back to the script.
func (b Blueprint) Instantiate(title string) (document.Document, error) {
	if err := b.Validate(); err != nil {
		return document.Document{}, err
	}
	created, err := document.New(title)
	if err != nil {
		return document.Document{}, err
	}
	created.Sections = instantiateSections(b.Sections)
	if err := created.Validate(); err != nil {
		return document.Document{}, err
	}
	return created, nil
}

func instantiateSections(sections []Section) []document.Section {
	instantiated := make([]document.Section, 0, len(sections))
	for _, section := range sections {
		instantiated = append(instantiated, document.Section{
			ID:            uuid.NewString(),
			DefinitionKey: strings.TrimSpace(section.Key),
			Title:         strings.TrimSpace(section.Title),
			Guidance:      section.Guidance,
			Required:      section.Required,
			// Every section starts with one empty rich text element so there is
			// somewhere to write without adding an element first.
			Elements: []document.Element{{ID: uuid.NewString(), Type: document.ElementRichText}},
			Sections: instantiateSections(section.Sections),
		})
	}
	return instantiated
}

// DefaultBlueprint is the ETP script the installation starts with. It is the
// same outline that used to be hardcoded in the document package, moved here so
// it can be published as revision 1 and edited from then on.
func DefaultBlueprint() Blueprint {
	text := []string{document.ElementRichText}
	return Blueprint{
		SchemaVersion: BlueprintSchemaVersion,
		DocumentType:  document.DocumentTypeETP,
		Sections: []Section{
			{
				Key:             "necessidade",
				Title:           "Necessidade da contratação",
				Guidance:        "Descreva o problema a resolver e o resultado esperado para a administração.",
				Required:        true,
				AllowedElements: text,
			},
			{
				Key:             "requisitos",
				Title:           "Requisitos da contratação",
				Guidance:        "Registre requisitos funcionais, técnicos e de sustentabilidade aplicáveis.",
				Required:        true,
				AllowedElements: text,
			},
			{
				Key:             "pesquisa-de-precos",
				Title:           "Pesquisa de preços",
				Guidance:        "Descreva fontes consultadas, datas, valores observados e o critério de cálculo.",
				Required:        true,
				AllowedElements: []string{document.ElementRichText, "priceResearch"},
			},
			{
				Key:             "mapa-de-riscos",
				Title:           "Mapa de riscos",
				Guidance:        "Relacione riscos, probabilidade, impacto, ações preventivas e de contingência.",
				Required:        true,
				AllowedElements: []string{document.ElementRichText, "riskMap"},
			},
			{
				Key:             "conclusao",
				Title:           "Conclusão",
				Guidance:        "Registre a conclusão sobre a viabilidade da contratação.",
				Required:        true,
				AllowedElements: text,
			},
		},
	}
}
