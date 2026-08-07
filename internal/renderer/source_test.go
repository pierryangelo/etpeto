package renderer

import (
	"strings"
	"testing"

	"github.com/etpeto/etpeto/internal/rendercontract"
)

func TestBuildSourceEscapesTypstSyntaxAndKeepsURLsAsText(t *testing.T) {
	request := rendercontract.Request{
		SchemaVersion:      1,
		TemplateRevisionID: "template-v1",
		Title:              `#import "evil.typ"`,
		Sections: []rendercontract.Section{{
			Level:      1,
			Title:      "Referências [externas]",
			Paragraphs: []string{`Consulte https://example.test e C:\temp`},
		}},
	}

	source, err := BuildSource(request)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(source, `#import "evil.typ"`) {
		t.Fatal("user text was emitted as executable Typst")
	}
	if !strings.Contains(source, "https://example.test") {
		t.Fatal("ordinary URL text was lost")
	}
	if !strings.Contains(source, `C:\\temp`) {
		t.Fatalf("backslash was not escaped in source: %s", source)
	}
}

func TestBuildSourceNumbersAndIndentsNestedSections(t *testing.T) {
	source, err := BuildSource(rendercontract.Request{
		SchemaVersion:      1,
		TemplateRevisionID: "template-v1",
		Title:              "Aquisição de notebooks",
		Sections: []rendercontract.Section{
			{Title: "Requisitos da contratação", Level: 1, Paragraphs: []string{"Introdução da seção."}},
			{Title: "Requisitos técnicos", Level: 2, Paragraphs: []string{"Detalhe técnico."}},
			{Title: "Desempenho mínimo", Level: 3},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Hierarchical numbering: a subsection prints as 2.1, not restarting at 1.
	if !strings.Contains(source, `#set heading(numbering: "1.1")`) {
		t.Fatal("headings are not numbered hierarchically")
	}
	for _, want := range []string{
		`#heading(level: 1, "Requisitos da contratação")`,
		`#heading(level: 2, "Requisitos técnicos")`,
		`#heading(level: 3, "Desempenho mínimo")`,
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("source is missing %s", want)
		}
	}
	// A subsection's own paragraph is indented to match its heading.
	if !strings.Contains(source, `#pad(left: 8pt, par("Detalhe técnico."))`) {
		t.Fatalf("nested paragraph was not indented:\n%s", source)
	}
	if !strings.Contains(source, `#par("Introdução da seção.")`) {
		t.Fatal("a top level paragraph should not be indented")
	}
}
