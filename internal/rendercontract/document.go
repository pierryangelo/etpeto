package rendercontract

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	CurrentSchemaVersion = 1
	MaxSections          = 100
	MaxTitleRunes        = 300
	// MaxSectionLevel matches the outline depth the document model accepts.
	MaxSectionLevel   = 4
	MaxParagraphRunes = 20_000
)

type Request struct {
	SchemaVersion      int       `json:"schemaVersion"`
	TemplateRevisionID string    `json:"templateRevisionId"`
	Title              string    `json:"title"`
	Sections           []Section `json:"sections"`
}

// Section carries its outline level so the template can number and indent
// headings. The list stays flat and in reading order: the renderer numbers by
// level, it does not rebuild the tree.
type Section struct {
	Title      string   `json:"title"`
	Level      int      `json:"level"`
	Paragraphs []string `json:"paragraphs"`
}

func (r Request) Validate() error {
	if r.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("schemaVersion: supported version is %d", CurrentSchemaVersion)
	}
	if strings.TrimSpace(r.TemplateRevisionID) == "" {
		return fmt.Errorf("templateRevisionId: must not be blank")
	}
	if err := validateTitle("title", r.Title); err != nil {
		return err
	}
	if len(r.Sections) == 0 || len(r.Sections) > MaxSections {
		return fmt.Errorf("sections: must contain between 1 and %d items", MaxSections)
	}
	for sectionIndex, section := range r.Sections {
		if err := validateTitle(fmt.Sprintf("sections[%d].title", sectionIndex), section.Title); err != nil {
			return err
		}
		if section.Level < 1 || section.Level > MaxSectionLevel {
			return fmt.Errorf(
				"sections[%d].level: must be between 1 and %d",
				sectionIndex,
				MaxSectionLevel,
			)
		}
		for paragraphIndex, paragraph := range section.Paragraphs {
			if utf8.RuneCountInString(paragraph) > MaxParagraphRunes {
				return fmt.Errorf(
					"sections[%d].paragraphs[%d]: paragraph exceeds %d characters",
					sectionIndex,
					paragraphIndex,
					MaxParagraphRunes,
				)
			}
		}
	}
	return nil
}

func validateTitle(field string, value string) error {
	length := utf8.RuneCountInString(value)
	if strings.TrimSpace(value) == "" || length > MaxTitleRunes {
		return fmt.Errorf("%s: must contain between 1 and %d characters", field, MaxTitleRunes)
	}
	return nil
}
