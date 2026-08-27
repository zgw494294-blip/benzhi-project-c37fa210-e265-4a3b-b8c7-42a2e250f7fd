package validation

import (
	"fmt"
	"strings"

	"stagecaption-finalizer/internal/domain"
)

func validateGlossary(r domain.CaptionRevision, terms []domain.GlossaryTerm, add findingAdder) {
	for _, cue := range r.Cues {
		for _, term := range terms {
			source, translated, needle, required := cue.SourceText, cue.TranslatedText, term.SourceText, term.RequiredTranslation
			if !term.CaseSensitive {
				source = strings.ToLower(source)
				translated = strings.ToLower(translated)
				needle = strings.ToLower(needle)
				required = strings.ToLower(required)
			}
			if strings.Contains(source, needle) && !strings.Contains(translated, required) {
				add("TERM_REQUIRED", domain.SeverityBlocking, cue.Sequence, "原文命中术语但译文未使用规定译法", fmt.Sprintf("source=%q required=%q", term.SourceText, term.RequiredTranslation))
			}
			for _, forbidden := range term.ForbiddenTranslations {
				candidate := forbidden
				if !term.CaseSensitive {
					candidate = strings.ToLower(candidate)
				}
				if candidate != "" && strings.Contains(translated, candidate) {
					add("TERM_FORBIDDEN", domain.SeverityBlocking, cue.Sequence, "译文包含禁用译法", fmt.Sprintf("source=%q forbidden=%q", term.SourceText, forbidden))
				}
			}
		}
	}
}
