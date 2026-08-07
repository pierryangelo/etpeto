package renderer

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/etpeto/etpeto/internal/rendercontract"
)

func BuildSource(request rendercontract.Request) (string, error) {
	if err := request.Validate(); err != nil {
		return "", err
	}

	var source strings.Builder
	source.WriteString("#set page(paper: \"a4\", margin: (x: 25mm, y: 22mm))\n")
	source.WriteString("#set text(lang: \"pt\", size: 11pt)\n")
	source.WriteString("#set par(justify: true, leading: 0.72em)\n")
	source.WriteString("#set heading(numbering: \"1.\")\n\n")
	fmt.Fprintf(&source, "#align(center, text(size: 18pt, weight: \"bold\", %s))\n", typstString(request.Title))
	source.WriteString("#v(18pt)\n#outline(title: [Sumário])\n#pagebreak()\n\n")

	for _, section := range request.Sections {
		fmt.Fprintf(&source, "#heading(level: 1, %s)\n", typstString(section.Title))
		for _, paragraph := range section.Paragraphs {
			fmt.Fprintf(&source, "#par(%s)\n", typstString(paragraph))
		}
		source.WriteByte('\n')
	}

	return source.String(), nil
}

func typstString(value string) string {
	var escaped strings.Builder
	escaped.WriteByte('"')
	for _, character := range value {
		switch character {
		case '\\':
			escaped.WriteString(`\\`)
		case '"':
			escaped.WriteString(`\"`)
		case '\n':
			escaped.WriteString(`\n`)
		case '\r':
			escaped.WriteString(`\r`)
		case '\t':
			escaped.WriteString(`\t`)
		default:
			if unicode.IsControl(character) {
				escaped.WriteString(`\u{`)
				escaped.WriteString(strconv.FormatInt(int64(character), 16))
				escaped.WriteByte('}')
				continue
			}
			escaped.WriteRune(character)
		}
	}
	escaped.WriteByte('"')
	return escaped.String()
}
