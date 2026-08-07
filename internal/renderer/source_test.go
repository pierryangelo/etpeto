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
