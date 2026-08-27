package application

import (
	"fmt"
	"sort"
	"stagecaption-finalizer/internal/domain"
	"strings"
)

type FindingSummary struct {
	Total    int `json:"total"`
	Blocking int `json:"blocking"`
	Open     int `json:"open"`
	Resolved int `json:"resolved"`
}

func SummarizeFindings(items []domain.ValidationFinding) FindingSummary {
	var s FindingSummary
	s.Total = len(items)
	for _, f := range items {
		if f.Severity == domain.SeverityBlocking {
			s.Blocking++
		}
		if f.Status == domain.FindingOpen {
			s.Open++
		} else {
			s.Resolved++
		}
	}
	return s
}
func FormatAudit(events []map[string]any) string {
	var b strings.Builder
	for _, e := range events {
		fmt.Fprintf(&b, "#%v %v [%v] %v\n", e["sequence"], e["type"], e["actor"], e["digest"])
	}
	return b.String()
}
func SortTerms(items []domain.GlossaryTerm) []domain.GlossaryTerm {
	out := append(make([]domain.GlossaryTerm, 0, len(items)), items...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].SourceText != out[j].SourceText {
			return out[i].SourceText < out[j].SourceText
		}
		return out[i].ID < out[j].ID
	})
	return out
}
func (a *Service) Summary(projectID string) (FindingSummary, error) {
	items, err := a.Findings(projectID)
	if err != nil {
		return FindingSummary{}, err
	}
	return SummarizeFindings(items), nil
}
