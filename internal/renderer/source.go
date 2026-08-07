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
	// "1.1" numbers every level from the outline position, so a subsection
	// prints as 2.1 rather than restarting at 1.
	source.WriteString("#set heading(numbering: \"1.1\")\n")
	// Each level is indented and set slightly smaller, which is what makes the
	// hierarchy readable on paper without relying on the number alone.
	source.WriteString("#show heading.where(level: 2): it => pad(left: 8pt, it)\n")
	source.WriteString("#show heading.where(level: 3): it => pad(left: 16pt, it)\n")
	source.WriteString("#show heading.where(level: 4): it => pad(left: 24pt, it)\n\n")
	fmt.Fprintf(&source, "#align(center, text(size: 18pt, weight: \"bold\", %s))\n", typstString(request.Title))
	source.WriteString("#v(18pt)\n#outline(title: [Sumário])\n#pagebreak()\n\n")

	for _, section := range request.Sections {
		level := section.Level
		if level < 1 {
			level = 1
		}
		fmt.Fprintf(&source, "#heading(level: %d, %s)\n", level, typstString(section.Title))
		// A section's own paragraphs are indented to match its heading, so a
		// parent's introduction reads as belonging to the parent.
		indent := (level - 1) * 8
		for _, paragraph := range section.Paragraphs {
			if indent > 0 {
				fmt.Fprintf(&source, "#pad(left: %dpt, par(%s))\n", indent, typstString(paragraph))
				continue
			}
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
