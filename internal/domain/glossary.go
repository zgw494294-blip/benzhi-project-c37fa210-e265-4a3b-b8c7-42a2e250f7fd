package domain

import (
	"fmt"
	"strings"
	"time"
)

func NewTerm(id, projectID, source, required string, forbidden []string, sensitive bool, version int64, now time.Time) (GlossaryTerm, error) {
	if strings.TrimSpace(source) == "" || strings.TrimSpace(required) == "" {
		return GlossaryTerm{}, fmt.Errorf("%w: 原词和规定译法不能为空", ErrInvalidInput)
	}
	seen := map[string]bool{}
	clean := make([]string, 0, len(forbidden))
	for _, item := range forbidden {
		item = strings.TrimSpace(item)
		if item != "" && !seen[item] {
			seen[item] = true
			clean = append(clean, item)
		}
	}
	for _, item := range clean {
		if equalTerm(item, strings.TrimSpace(required), sensitive) {
			return GlossaryTerm{}, fmt.Errorf("%w: 规定译法不能同时列为禁用译法", ErrInvalidInput)
		}
	}
	return GlossaryTerm{ID: id, ProjectID: projectID, SourceText: strings.TrimSpace(source), RequiredTranslation: strings.TrimSpace(required), ForbiddenTranslations: clean, CaseSensitive: sensitive, Version: version, UpdatedAt: now.UTC()}, nil
}

func equalTerm(a, b string, sensitive bool) bool {
	if sensitive {
		return a == b
	}
	return strings.EqualFold(a, b)
}
