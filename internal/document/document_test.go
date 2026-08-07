package document

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestNewFromDefaultOutlineCreatesTheApprovedETPSections(t *testing.T) {
	created, err := New("Aquisição de notebooks")
	if err != nil {
		t.Fatal(err)
	}
	if created.SchemaVersion != SchemaVersion || created.DocumentType != DocumentTypeETP {
		t.Fatalf("document envelope = %#v", created)
	}
	want := []string{"necessidade", "requisitos", "pesquisa-de-precos", "mapa-de-riscos", "conclusao"}
	if len(created.Sections) != len(want) {
		t.Fatalf("sections = %d, want %d", len(created.Sections), len(want))
	}
	for index, key := range want {
		section := created.Sections[index]
		if section.DefinitionKey != key {
			t.Fatalf("section %d key = %q, want %q", index, section.DefinitionKey, key)
		}
		if section.ID == "" || section.Title == "" {
			t.Fatalf("section %d = %#v", index, section)
		}
		if len(section.Elements) != 1 || section.Elements[0].Type != ElementRichText {
			t.Fatalf("section %d elements = %#v", index, section.Elements)
		}
	}
}

func TestNewGivesEverySectionAndElementADistinctIdentifier(t *testing.T) {
	created, err := New("Aquisição de notebooks")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, section := range created.Sections {
		if seen[section.ID] {
			t.Fatalf("duplicate section identifier %q", section.ID)
		}
		seen[section.ID] = true
		for _, element := range section.Elements {
			if seen[element.ID] {
				t.Fatalf("duplicate element identifier %q", element.ID)
			}
			seen[element.ID] = true
		}
	}
}

func TestNewRejectsABlankTitle(t *testing.T) {
	if _, err := New("   "); !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("New() error = %v, want ErrInvalidDocument", err)
	}
}

func TestValidateRejectsContentOutsideTheAcceptedLimits(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Document)
	}{
		{
			name:   "unknown schema version",
			mutate: func(d *Document) { d.SchemaVersion = 99 },
		},
		{
			name:   "unknown document type",
			mutate: func(d *Document) { d.DocumentType = "contrato" },
		},
		{
			name:   "blank title",
			mutate: func(d *Document) { d.Title = "  " },
		},
		{
			name:   "oversized title",
			mutate: func(d *Document) { d.Title = strings.Repeat("á", maxTitleRunes+1) },
		},
		{
			name:   "no sections",
			mutate: func(d *Document) { d.Sections = nil },
		},
		{
			name:   "section without identifier",
			mutate: func(d *Document) { d.Sections[0].ID = "" },
		},
		{
			name:   "duplicate section identifier",
			mutate: func(d *Document) { d.Sections[1].ID = d.Sections[0].ID },
		},
		{
			name:   "unknown element type",
			mutate: func(d *Document) { d.Sections[0].Elements[0].Type = "planilha" },
		},
		{
			name: "rich text above the accepted size",
			mutate: func(d *Document) {
				d.Sections[0].Elements[0].RichText = json.RawMessage(
					`{"root":{"children":[],"note":"` + strings.Repeat("x", maxRichTextBytes) + `"}}`,
				)
			},
		},
		{
			name: "malformed rich text",
			mutate: func(d *Document) {
				d.Sections[0].Elements[0].RichText = json.RawMessage(`{"root":`)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			created, err := New("Aquisição de notebooks")
			if err != nil {
				t.Fatal(err)
			}
			test.mutate(&created)
			if err := created.Validate(); !errors.Is(err, ErrInvalidDocument) {
				t.Fatalf("Validate() error = %v, want ErrInvalidDocument", err)
			}
		})
	}
}

func TestValidateAcceptsAFilledDocument(t *testing.T) {
	created, err := New("Aquisição de notebooks")
	if err != nil {
		t.Fatal(err)
	}
	created.Sections[0].Elements[0].RichText = lexicalState("Necessidade descrita pela equipe.")
	if err := created.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestDocumentRoundTripsThroughJSONWithoutLoss(t *testing.T) {
	created, err := New("Aquisição de notebooks")
	if err != nil {
		t.Fatal(err)
	}
	created.Sections[2].Elements[0].RichText = lexicalState("Três fornecedores consultados.")

	encoded, err := json.Marshal(created)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Document
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Title != created.Title || len(decoded.Sections) != len(created.Sections) {
		t.Fatalf("decoded document = %#v", decoded)
	}
	for index, section := range created.Sections {
		if decoded.Sections[index].ID != section.ID {
			t.Fatalf("section %d identifier changed", index)
		}
		original := string(section.Elements[0].RichText)
		restored := string(decoded.Sections[index].Elements[0].RichText)
		if original != "" && restored != original {
			t.Fatalf("rich text %d changed:\n%s\n%s", index, original, restored)
		}
	}
}

func lexicalState(text string) json.RawMessage {
	state := map[string]any{
		"root": map[string]any{
			"type":    "root",
			"version": 1,
			"children": []any{
				map[string]any{
					"type":    "paragraph",
					"version": 1,
					"children": []any{
						map[string]any{"type": "text", "version": 1, "text": text},
					},
				},
			},
		},
	}
	encoded, err := json.Marshal(state)
	if err != nil {
		panic(err)
	}
	return encoded
}
