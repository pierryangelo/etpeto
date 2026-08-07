package document

import "strconv"

// maxOutlineDepth caps nesting at four levels. Beyond "1.1.1.1" a numbered
// heading stops helping the reader find anything, and the indentation no longer
// fits a side panel.
const maxOutlineDepth = 4

// Flatten walks the outline in reading order — each section immediately
// followed by its own subsections — and pairs every section with its number and
// depth. This is what the outline panel, the progress count and the renderer
// all iterate over, so they cannot disagree about the order.
func (d Document) Flatten() []FlatSection {
	flattened := make([]FlatSection, 0, len(d.Sections))
	appendSections(&flattened, d.Sections, "", 1)
	return flattened
}

func appendSections(into *[]FlatSection, sections []Section, prefix string, depth int) {
	for index := range sections {
		section := &sections[index]
		number := strconv.Itoa(index + 1)
		if prefix != "" {
			number = prefix + "." + number
		}
		*into = append(*into, FlatSection{Section: section, Number: number, Depth: depth})
		appendSections(into, section.Sections, number, depth+1)
	}
}

// FindSection resolves an identifier at any depth. Callers hold section
// identifiers, never paths, so moving a section does not invalidate a link.
func (d Document) FindSection(id string) *Section {
	for _, entry := range d.Flatten() {
		if entry.Section.ID == id {
			return entry.Section
		}
	}
	return nil
}
