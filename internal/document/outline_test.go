package document

import (
	"encoding/json"
	"errors"
	"testing"
)

func nestedDocument(t *testing.T) Document {
	t.Helper()
	return Document{
		SchemaVersion: SchemaVersion,
		DocumentType:  DocumentTypeETP,
		Title:         "Aquisição de notebooks",
		Sections: []Section{
			{
				ID:            "necessidade",
				DefinitionKey: "necessidade",
				Title:         "Necessidade da contratação",
				Required:      true,
				Elements:      []Element{{ID: "e1", Type: ElementRichText}},
			},
			{
				ID:            "requisitos",
				DefinitionKey: "requisitos",
				Title:         "Requisitos da contratação",
				Required:      true,
				// A section with subsections still writes: the introductory
				// paragraph belongs to the parent, not to an invented child.
				Elements: []Element{{ID: "e2", Type: ElementRichText}},
				Sections: []Section{
					{
						ID:            "tecnicos",
						DefinitionKey: "requisitos-tecnicos",
						Title:         "Requisitos técnicos",
						Required:      true,
						Elements:      []Element{{ID: "e3", Type: ElementRichText}},
						Sections: []Section{
							{
								ID:            "desempenho",
								DefinitionKey: "requisitos-desempenho",
								Title:         "Desempenho mínimo",
								Required:      false,
								Elements:      []Element{{ID: "e4", Type: ElementRichText}},
							},
						},
					},
					{
						ID:            "sustentabilidade",
						DefinitionKey: "requisitos-sustentabilidade",
						Title:         "Requisitos de sustentabilidade",
						Required:      false,
						Elements:      []Element{{ID: "e5", Type: ElementRichText}},
					},
				},
			},
		},
	}
}

func TestFlattenNumbersSectionsHierarchically(t *testing.T) {
	flattened := nestedDocument(t).Flatten()

	want := []struct {
		number string
		depth  int
		title  string
	}{
		{"1", 1, "Necessidade da contratação"},
		{"2", 1, "Requisitos da contratação"},
		{"2.1", 2, "Requisitos técnicos"},
		{"2.1.1", 3, "Desempenho mínimo"},
		{"2.2", 2, "Requisitos de sustentabilidade"},
	}
	if len(flattened) != len(want) {
		t.Fatalf("flattened = %d sections, want %d", len(flattened), len(want))
	}
	for index, expected := range want {
		got := flattened[index]
		if got.Number != expected.number || got.Depth != expected.depth {
			t.Fatalf("section %d = %q depth %d, want %q depth %d",
				index, got.Number, got.Depth, expected.number, expected.depth)
		}
		if got.Section.Title != expected.title {
			t.Fatalf("section %d title = %q, want %q", index, got.Section.Title, expected.title)
		}
	}
}

func TestFlattenKeepsDocumentOrderForNavigation(t *testing.T) {
	flattened := nestedDocument(t).Flatten()

	// Depth-first order is the reading order: a subsection comes right after
	// its parent, never after the parent's next sibling.
	var numbers []string
	for _, entry := range flattened {
		numbers = append(numbers, entry.Number)
	}
	expected := []string{"1", "2", "2.1", "2.1.1", "2.2"}
	for index, number := range expected {
		if numbers[index] != number {
			t.Fatalf("order = %v, want %v", numbers, expected)
		}
	}
}

func TestFindSectionReachesAnyDepth(t *testing.T) {
	document := nestedDocument(t)

	found := document.FindSection("desempenho")
	if found == nil || found.Title != "Desempenho mínimo" {
		t.Fatalf("FindSection() = %#v", found)
	}
	if document.FindSection("inexistente") != nil {
		t.Fatal("unknown identifier returned a section")
	}
}

func TestEachSectionDeclaresItsOwnObligation(t *testing.T) {
	document := nestedDocument(t)

	// A required parent does not force its children: the script says exactly
	// what is demanded, at every level.
	requirements := map[string]bool{}
	for _, entry := range document.Flatten() {
		requirements[entry.Section.ID] = entry.Section.Required
	}
	if !requirements["requisitos"] || !requirements["tecnicos"] {
		t.Fatalf("required sections = %#v", requirements)
	}
	if requirements["sustentabilidade"] || requirements["desempenho"] {
		t.Fatalf("optional sections were treated as required: %#v", requirements)
	}
}

func TestValidateRejectsNestingBeyondTheAcceptedDepth(t *testing.T) {
	document := nestedDocument(t)
	deepest := &document.Sections[1].Sections[0].Sections[0]
	deepest.Sections = []Section{{
		ID:            "nivel-4",
		DefinitionKey: "nivel-4",
		Title:         "Quarto nível",
		Elements:      []Element{{ID: "e6", Type: ElementRichText}},
		Sections: []Section{{
			ID:            "nivel-5",
			DefinitionKey: "nivel-5",
			Title:         "Quinto nível",
		}},
	}}

	if err := document.Validate(); !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("Validate() error = %v, want ErrInvalidDocument", err)
	}
}

func TestValidateRejectsAnIdentifierReusedAcrossLevels(t *testing.T) {
	document := nestedDocument(t)
	document.Sections[1].Sections[1].ID = document.Sections[0].ID

	if err := document.Validate(); !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("Validate() error = %v, want ErrInvalidDocument", err)
	}
}

func TestValidateAcceptsTheNestedOutline(t *testing.T) {
	if err := nestedDocument(t).Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestNestedSectionsSurviveJSONRoundTrip(t *testing.T) {
	original := nestedDocument(t)

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Document
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Flatten()) != len(original.Flatten()) {
		t.Fatalf("round trip changed the outline: %d sections", len(decoded.Flatten()))
	}
	if decoded.Sections[1].Sections[0].Sections[0].Title != "Desempenho mínimo" {
		t.Fatal("a nested subsection was lost")
	}
}
