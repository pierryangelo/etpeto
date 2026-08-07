package rendercontract

import (
	"strings"
	"testing"
)

func TestRequestValidateAcceptsVersionOne(t *testing.T) {
	request := Request{
		SchemaVersion:      1,
		TemplateRevisionID: "template-v1",
		Title:              "Estudo técnico preliminar",
		Sections: []Section{{
			Level:      1,
			Title:      "Necessidade da contratação",
			Paragraphs: []string{"Texto da necessidade."},
		}},
	}

	if err := request.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestRequestValidateRejectsUnsupportedVersion(t *testing.T) {
	request := validRequest()
	request.SchemaVersion = 2

	if err := request.Validate(); err == nil || !strings.Contains(err.Error(), "schemaVersion") {
		t.Fatalf("Validate() error = %v, want schemaVersion error", err)
	}
}

func TestRequestValidateRejectsOversizedParagraph(t *testing.T) {
	request := validRequest()
	request.Sections[0].Paragraphs = []string{strings.Repeat("á", 20_001)}

	if err := request.Validate(); err == nil || !strings.Contains(err.Error(), "paragraph") {
		t.Fatalf("Validate() error = %v, want paragraph error", err)
	}
}

func validRequest() Request {
	return Request{
		SchemaVersion:      1,
		TemplateRevisionID: "template-v1",
		Title:              "ETP",
		Sections: []Section{
			{Title: "Seção", Level: 1, Paragraphs: []string{"Texto"}},
		},
	}
}
