//go:build integration

package renderer

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/etpeto/etpeto/internal/rendercontract"
)

func TestCLIRunnerProducesPDFWithRealTypst(t *testing.T) {
	source, err := BuildSource(rendercontract.Request{
		SchemaVersion:      1,
		TemplateRevisionID: "template-v1",
		Title:              `ETP #1 "seguro"`,
		Sections: []rendercontract.Section{{
			Level:      1,
			Title:      "Necessidade [principal]",
			Paragraphs: []string{`Consulte https://example.test e C:\temp.`},
		}},
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
		t.Fatalf("Render() error = %v\nsource:\n%s", err, source)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("rendered bytes do not start with PDF signature: %q", pdf[:min(len(pdf), 16)])
	}
}
