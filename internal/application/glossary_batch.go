package application

import (
	"fmt"
	"sort"
	"strings"

	"stagecaption-finalizer/internal/domain"
	"stagecaption-finalizer/internal/store"
)

type GlossaryConflict struct {
	Line    int    `json:"line"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type GlossaryConflictError struct {
	Conflicts []GlossaryConflict `json:"conflicts"`
}

func (e *GlossaryConflictError) Error() string {
	return fmt.Sprintf("术语批量导入存在 %d 项冲突", len(e.Conflicts))
}
func (e *GlossaryConflictError) Unwrap() error { return domain.ErrInvalidInput }

func (a *Service) ImportGlossary(projectID string, c BatchGlossaryCommand) (BatchGlossaryResult, error) {
	if err := requireKey(c.IdempotencyKey); err != nil {
		return BatchGlossaryResult{}, err
	}
	if len(c.Entries) == 0 {
		return BatchGlossaryResult{}, fmt.Errorf("%w: entries 不能为空", domain.ErrInvalidInput)
	}
	key := idemKey(projectID, "glossary-batch", c.IdempotencyKey)
	var out BatchGlossaryResult
	err := a.repo.Update(projectID, "glossary.updated", c.Actor, func(s *store.Snapshot) error {
		if old, ok, err := replay[BatchGlossaryResult](s, key); err != nil {
			return err
		} else if ok {
			out = old
			return nil
		}
		p, ok := s.Projects[projectID]
		if !ok {
			return domain.ErrNotFound
		}
		if err := p.CheckVersion(c.ExpectedVersion); err != nil {
			return err
		}
		if err := p.EnsureMutable(); err != nil {
			return err
		}
		if p.Status == domain.StatusInReview || p.Status == domain.StatusApproved {
			return fmt.Errorf("%w: 复核或批准阶段术语快照不可修改", domain.ErrInvalidState)
		}

		nextVersion := p.GlossaryVersion + 1
		now := a.now()
		current := currentTerms(*s, p)
		seen := make(map[string]domain.GlossaryTerm, len(current)+len(c.Entries))
		for _, term := range current {
			seen[strings.ToLower(strings.TrimSpace(term.SourceText))] = term
		}
		imported := make([]domain.GlossaryTerm, 0, len(c.Entries))
		conflicts := make([]GlossaryConflict, 0)
		for i, entry := range c.Entries {
			line := i + 1
			term, err := domain.NewTerm(newID("term"), projectID, entry.SourceText, entry.RequiredTranslation, entry.ForbiddenTranslations, entry.CaseSensitive, nextVersion, now)
			if err != nil {
				conflicts = append(conflicts, GlossaryConflict{Line: line, Code: "INVALID_TERM", Message: err.Error()})
				continue
			}
			key := strings.ToLower(term.SourceText)
			if previous, exists := seen[key]; exists {
				code := "DUPLICATE_SOURCE"
				message := "原词与当前快照或本批次重复"
				if previous.CaseSensitive != term.CaseSensitive {
					code = "CASE_POLICY_CONFLICT"
					message = "同一原词的大小写策略冲突"
				}
				conflicts = append(conflicts, GlossaryConflict{Line: line, Code: code, Message: message})
				continue
			}
			seen[key] = term
			imported = append(imported, term)
		}
		if len(conflicts) > 0 {
			return &GlossaryConflictError{Conflicts: conflicts}
		}

		next := make([]domain.GlossaryTerm, 0, len(current)+len(imported))
		for _, term := range current {
			term.Version = nextVersion
			term.UpdatedAt = now.UTC()
			next = append(next, term)
		}
		next = append(next, imported...)
		sort.SliceStable(next, func(i, j int) bool {
			left, right := strings.ToLower(next[i].SourceText), strings.ToLower(next[j].SourceText)
			if left != right {
				return left < right
			}
			return next[i].ID < next[j].ID
		})
		s.Terms[projectID] = append(s.Terms[projectID], next...)
		p.GlossaryVersion = nextVersion
		repointCurrentRevision(s, &p, now)
		p.Touch(now)
		s.Projects[projectID] = p
		out = BatchGlossaryResult{GlossaryVersion: nextVersion, ImportedCount: len(imported), TotalCount: len(next), Entries: imported}
		return remember(s, key, "glossary-batch", out, now)
	})
	return out, err
}
