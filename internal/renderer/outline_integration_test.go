//go:build integration

package renderer

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/etpeto/etpeto/internal/rendercontract"
)

// A four level outline has to survive the real Typst binary, not only the
// source builder: numbering and indentation are template behaviour.
func TestNestedOutlineRendersWithRealTypst(t *testing.T) {
	source, err := BuildSource(rendercontract.Request{
		SchemaVersion:      1,
		TemplateRevisionID: "t1",
		Title:              "Aquisição de notebooks",
		Sections: []rendercontract.Section{
			{Title: "Necessidade da contratação", Level: 1, Paragraphs: []string{"Equipamentos com mais de cinco anos."}},
			{Title: "Requisitos da contratação", Level: 1, Paragraphs: []string{"Introdução aos requisitos."}},
			{Title: "Requisitos técnicos", Level: 2, Paragraphs: []string{"Processador e memória mínimos."}},
			{Title: "Desempenho mínimo", Level: 3, Paragraphs: []string{"Índice de referência exigido."}},
			{Title: "Requisitos de sustentabilidade", Level: 2, Paragraphs: []string{"Eficiência energética verificável."}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	runner, err := NewCLIRunner(t.TempDir(), 1, 15*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	pdf, err := runner.Render(context.Background(), source)
	if err != nil {
		t.Fatalf("Render() error = %v\n%s", err, source)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatal("nested outline did not produce a PDF")
	}
}
