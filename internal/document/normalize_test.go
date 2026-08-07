package document

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalizeExtractsOneParagraphPerLexicalBlock(t *testing.T) {
	created := documentWithRichText(t, `{
	  "root": {
	    "type": "root",
	    "children": [
	      {"type": "heading", "tag": "h2", "children": [{"type": "text", "text": "Contexto"}]},
	      {"type": "paragraph", "children": [
	        {"type": "text", "text": "A equipe "},
	        {"type": "text", "text": "identificou a necessidade."}
	      ]},
	      {"type": "list", "children": [
	        {"type": "listitem", "children": [{"type": "text", "text": "Primeiro requisito"}]},
	        {"type": "listitem", "children": [{"type": "text", "text": "Segundo requisito"}]}
	      ]}
	    ]
	  }
	}`)

	request := Normalize(created, "template-v1")
	if err := request.Validate(); err != nil {
		t.Fatalf("normalized request is not renderable: %v", err)
	}
	want := []string{
		"Contexto",
		"A equipe identificou a necessidade.",
		"Primeiro requisito",
		"Segundo requisito",
	}
	got := request.Sections[0].Paragraphs
	if len(got) != len(want) {
		t.Fatalf("paragraphs = %#v, want %#v", got, want)
	}
	for index, paragraph := range want {
		if got[index] != paragraph {
			t.Fatalf("paragraph %d = %q, want %q", index, got[index], paragraph)
		}
	}
}

func TestNormalizeKeepsEmptySectionsSoTheOutlineStaysVisibleInThePDF(t *testing.T) {
	created, err := New("Aquisição de notebooks")
	if err != nil {
		t.Fatal(err)
	}

	request := Normalize(created, "template-v1")
	if err := request.Validate(); err != nil {
		t.Fatalf("empty document is not renderable: %v", err)
	}
	if len(request.Sections) != len(created.Sections) {
		t.Fatalf("sections = %d, want %d", len(request.Sections), len(created.Sections))
	}
	if request.Title != "Aquisição de notebooks" || request.SchemaVersion != 1 {
		t.Fatalf("request envelope = %#v", request)
	}
	if len(request.Sections[0].Paragraphs) != 0 {
		t.Fatalf("empty section carried paragraphs: %#v", request.Sections[0].Paragraphs)
	}
}

func TestNormalizeIgnoresMalformedOrEmptyStateWithoutFailing(t *testing.T) {
	tests := []string{`{"root":`, `{}`, `{"root":{"children":[]}}`, `{"root":{"children":[{"type":"paragraph","children":[]}]}}`}
	for _, state := range tests {
		t.Run(state, func(t *testing.T) {
			created := documentWithRichText(t, state)
			request := Normalize(created, "template-v1")
			if len(request.Sections[0].Paragraphs) != 0 {
				t.Fatalf("paragraphs = %#v, want none", request.Sections[0].Paragraphs)
			}
		})
	}
}

func TestNormalizeTruncatesParagraphsToTheRendererLimit(t *testing.T) {
	long := strings.Repeat("á", 25_000)
	created := documentWithRichText(t, `{"root":{"children":[{"type":"paragraph","children":[{"type":"text","text":"`+long+`"}]}]}}`)

	request := Normalize(created, "template-v1")
	if err := request.Validate(); err != nil {
		t.Fatalf("oversized paragraph was not truncated: %v", err)
	}
}

func documentWithRichText(t *testing.T, state string) Document {
	t.Helper()
	created, err := New("Aquisição de notebooks")
	if err != nil {
		t.Fatal(err)
	}
	created.Sections[0].Elements[0].RichText = json.RawMessage(state)
	return created
}
