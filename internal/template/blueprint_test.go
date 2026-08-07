package template

import (
	"errors"
	"strings"
	"testing"

	"github.com/etpeto/etpeto/internal/document"
)

func nestedBlueprint() Blueprint {
	text := []string{document.ElementRichText}
	return Blueprint{
		SchemaVersion: BlueprintSchemaVersion,
		DocumentType:  document.DocumentTypeETP,
		Sections: []Section{
			{
				Key:             "necessidade",
				Title:           "Necessidade da contratação",
				Guidance:        "Descreva o problema a resolver.",
				Required:        true,
				AllowedElements: text,
			},
			{
				Key:             "requisitos",
				Title:           "Requisitos da contratação",
				Required:        true,
				AllowRename:     true,
				AllowedElements: text,
				Sections: []Section{
					{
						Key:             "requisitos-tecnicos",
						Title:           "Requisitos técnicos",
						Required:        true,
						AllowedElements: text,
					},
					{
						Key:             "requisitos-sustentabilidade",
						Title:           "Sustentabilidade",
						Required:        false,
						AllowedElements: text,
					},
				},
			},
		},
	}
}

func TestDefaultBlueprintIsValidAndProducesTheETPOutline(t *testing.T) {
	blueprint := DefaultBlueprint()
	if err := blueprint.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	created, err := blueprint.Instantiate("Aquisição de notebooks")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"necessidade", "requisitos", "pesquisa-de-precos", "mapa-de-riscos", "conclusao"}
	if len(created.Sections) != len(want) {
		t.Fatalf("sections = %d, want %d", len(created.Sections), len(want))
	}
	for index, key := range want {
		if created.Sections[index].DefinitionKey != key {
			t.Fatalf("section %d key = %q, want %q", index, created.Sections[index].DefinitionKey, key)
		}
	}
}

func TestInstantiateCarriesGuidanceObligationAndNesting(t *testing.T) {
	created, err := nestedBlueprint().Instantiate("Aquisição de notebooks")
	if err != nil {
		t.Fatal(err)
	}

	first := created.Sections[0]
	if first.Guidance != "Descreva o problema a resolver." || !first.Required {
		t.Fatalf("first section = %#v", first)
	}
	requisitos := created.Sections[1]
	if len(requisitos.Sections) != 2 {
		t.Fatalf("subsections = %d, want 2", len(requisitos.Sections))
	}
	if requisitos.Sections[1].Required {
		t.Fatal("an optional subsection was instantiated as required")
	}
	// A section with subsections still holds its own content.
	if len(requisitos.Elements) != 1 {
		t.Fatalf("parent section elements = %#v", requisitos.Elements)
	}
	if flattened := created.Flatten(); len(flattened) != 4 {
		t.Fatalf("flattened = %d sections, want 4", len(flattened))
	}
}

func TestInstantiateMintsFreshIdentifiersForEachDocument(t *testing.T) {
	blueprint := nestedBlueprint()

	first, err := blueprint.Instantiate("Primeiro ETP")
	if err != nil {
		t.Fatal(err)
	}
	second, err := blueprint.Instantiate("Segundo ETP")
	if err != nil {
		t.Fatal(err)
	}

	// Two documents from one revision must not share identifiers, or editing
	// one would address elements of the other.
	used := map[string]bool{}
	for _, entry := range first.Flatten() {
		used[entry.Section.ID] = true
		for _, element := range entry.Section.Elements {
			used[element.ID] = true
		}
	}
	for _, entry := range second.Flatten() {
		if used[entry.Section.ID] {
			t.Fatalf("section identifier %q was reused across documents", entry.Section.ID)
		}
		for _, element := range entry.Section.Elements {
			if used[element.ID] {
				t.Fatalf("element identifier %q was reused across documents", element.ID)
			}
		}
	}
	// The meaning, however, is shared.
	if first.Sections[0].DefinitionKey != second.Sections[0].DefinitionKey {
		t.Fatal("section keys diverged between documents of the same script")
	}
}

func TestValidateRejectsScriptsThatWouldProduceAnUnusableDocument(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Blueprint)
	}{
		{"unknown schema version", func(b *Blueprint) { b.SchemaVersion = 9 }},
		{"unknown document type", func(b *Blueprint) { b.DocumentType = "contrato" }},
		{"no sections", func(b *Blueprint) { b.Sections = nil }},
		{"blank key", func(b *Blueprint) { b.Sections[0].Key = "  " }},
		{"blank title", func(b *Blueprint) { b.Sections[0].Title = "" }},
		{
			name:   "oversized guidance",
			mutate: func(b *Blueprint) { b.Sections[0].Guidance = strings.Repeat("á", maxGuidanceRunes+1) },
		},
		{
			name: "key repeated across levels",
			mutate: func(b *Blueprint) {
				b.Sections[1].Sections[0].Key = b.Sections[0].Key
			},
		},
		{
			name:   "section that accepts nothing",
			mutate: func(b *Blueprint) { b.Sections[0].AllowedElements = nil },
		},
		{
			name:   "unsupported element type",
			mutate: func(b *Blueprint) { b.Sections[0].AllowedElements = []string{"planilha"} },
		},
		{
			// requisitos(1) > técnicos(2) > n3 > n4 > n5 exceeds the four
			// levels a numbered heading still helps anyone read.
			name: "nesting beyond the accepted depth",
			mutate: func(b *Blueprint) {
				text := []string{document.ElementRichText}
				fifth := Section{Key: "n5", Title: "Nível 5", AllowedElements: text}
				fourth := Section{Key: "n4", Title: "Nível 4", AllowedElements: text, Sections: []Section{fifth}}
				third := Section{Key: "n3", Title: "Nível 3", AllowedElements: text, Sections: []Section{fourth}}
				b.Sections[1].Sections[0].Sections = []Section{third}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			blueprint := nestedBlueprint()
			test.mutate(&blueprint)

			if err := blueprint.Validate(); !errors.Is(err, ErrInvalidBlueprint) {
				t.Fatalf("Validate() error = %v, want ErrInvalidBlueprint", err)
			}
		})
	}
}

func TestInstantiateRefusesAnInvalidScriptBeforeCreatingAnything(t *testing.T) {
	blueprint := nestedBlueprint()
	blueprint.Sections[0].Key = ""

	if _, err := blueprint.Instantiate("Aquisição de notebooks"); !errors.Is(err, ErrInvalidBlueprint) {
		t.Fatalf("Instantiate() error = %v, want ErrInvalidBlueprint", err)
	}
}

func TestInstantiateRefusesABlankDocumentTitle(t *testing.T) {
	if _, err := nestedBlueprint().Instantiate("   "); !errors.Is(err, document.ErrInvalidDocument) {
		t.Fatalf("Instantiate() error = %v, want ErrInvalidDocument", err)
	}
}
