package document

import (
	"encoding/json"
	"strings"

	"github.com/etpeto/etpeto/internal/rendercontract"
)

// blockNodes are the Lexical node types that become one paragraph in the PDF.
// Anything else is traversed for its text but does not open a new block.
var blockNodes = map[string]bool{
	"paragraph": true,
	"heading":   true,
	"quote":     true,
	"listitem":  true,
}

type lexicalNode struct {
	Type     string        `json:"type"`
	Text     string        `json:"text"`
	Children []lexicalNode `json:"children"`
}

// Normalize converts the canonical document into the closed renderer contract.
// It never fails: a malformed or empty editor state yields no paragraphs, so a
// damaged element can never block the preview of the rest of the document.
func Normalize(source Document, templateRevisionID string) rendercontract.Request {
	sections := make([]rendercontract.Section, 0, len(source.Sections))
	for _, section := range source.Sections {
		paragraphs := make([]string, 0)
		for _, element := range section.Elements {
			if element.Type != ElementRichText {
				continue
			}
			paragraphs = append(paragraphs, extractParagraphs(element.RichText)...)
		}
		sections = append(sections, rendercontract.Section{
			Title:      section.Title,
			Paragraphs: paragraphs,
		})
	}
	return rendercontract.Request{
		SchemaVersion:      rendercontract.CurrentSchemaVersion,
		TemplateRevisionID: templateRevisionID,
		Title:              source.Title,
		Sections:           sections,
	}
}

func extractParagraphs(state json.RawMessage) []string {
	if len(state) == 0 {
		return nil
	}
	var envelope struct {
		Root lexicalNode `json:"root"`
	}
	if err := json.Unmarshal(state, &envelope); err != nil {
		return nil
	}
	var paragraphs []string
	collectParagraphs(envelope.Root, &paragraphs)
	return paragraphs
}

func collectParagraphs(node lexicalNode, into *[]string) {
	if blockNodes[node.Type] {
		if text := strings.TrimSpace(collectText(node)); text != "" {
			*into = append(*into, truncateRunes(text, rendercontract.MaxParagraphRunes))
		}
		// A list item may nest another list; those blocks are collected below.
	}
	for _, child := range node.Children {
		if blockNodes[node.Type] && !blockNodes[child.Type] {
			continue
		}
		collectParagraphs(child, into)
	}
}

func collectText(node lexicalNode) string {
	var builder strings.Builder
	builder.WriteString(node.Text)
	for _, child := range node.Children {
		if blockNodes[child.Type] {
			continue
		}
		builder.WriteString(collectText(child))
	}
	return builder.String()
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
