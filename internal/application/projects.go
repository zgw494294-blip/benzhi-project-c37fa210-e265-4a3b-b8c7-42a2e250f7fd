package application

import (
	"fmt"
	"sort"
	"stagecaption-finalizer/internal/domain"
	"stagecaption-finalizer/internal/store"
	"strings"
)

func (a *Service) CreateProject(c CreateProjectCommand) (domain.CaptionProject, error) {
	if err := requireKey(c.IdempotencyKey); err != nil {
		return domain.CaptionProject{}, err
	}
	key := idemKey("_", "create", c.IdempotencyKey)
	var out domain.CaptionProject
	err := a.repo.Update("_create", "project.created", c.Actor, func(s *store.Snapshot) error {
		if old, ok, err := replay[domain.CaptionProject](s, key); err != nil {
			return err
		} else if ok {
			out = old
			return nil
		}
		p, err := domain.NewProject(newID("project"), c.Title, c.SourceLanguage, c.TargetLanguage, c.FrameRate, c.MinDisplayMillis, c.MaxDisplayMillis, a.now())
		if err != nil {
			return err
		}
		s.Projects[p.ID] = *p
		out = *p
		return remember(s, key, "create", out, a.now())
	})
	return out, err
}

func (a *Service) AddTerm(projectID string, c TermCommand) (domain.GlossaryTerm, error) {
	if err := requireKey(c.IdempotencyKey); err != nil {
		return domain.GlossaryTerm{}, err
	}
	key := idemKey(projectID, "term", c.IdempotencyKey)
	var out domain.GlossaryTerm
	err := a.repo.Update(projectID, "glossary.updated", c.Actor, func(s *store.Snapshot) error {
		if old, ok, err := replay[domain.GlossaryTerm](s, key); err != nil {
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
			return fmt.Errorf("%w: 复核阶段术语快照不可修改", domain.ErrInvalidState)
		}
		next := p.GlossaryVersion + 1
		old := currentTerms(*s, p)
		for i := range old {
			old[i].Version = next
		}
		term, err := domain.NewTerm(newID("term"), projectID, c.SourceText, c.RequiredTranslation, c.ForbiddenTranslations, c.CaseSensitive, next, a.now())
		if err != nil {
			return err
		}
		old = append(old, term)
		sort.Slice(old, func(i, j int) bool { return strings.ToLower(old[i].SourceText) < strings.ToLower(old[j].SourceText) })
		s.Terms[projectID] = append(s.Terms[projectID], old...)
		p.GlossaryVersion = next
		p.Touch(a.now())
		s.Projects[projectID] = p
		out = term
		return remember(s, key, "term", out, a.now())
	})
	return out, err
}
