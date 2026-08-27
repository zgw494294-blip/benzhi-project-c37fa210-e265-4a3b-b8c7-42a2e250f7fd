package application

import (
	"fmt"
	"strings"

	"stagecaption-finalizer/internal/domain"
	"stagecaption-finalizer/internal/store"
)

func (a *Service) ResolveFindings(projectID string, c BatchResolveCommand) (BatchResolveResult, error) {
	if err := requireKey(c.IdempotencyKey); err != nil {
		return BatchResolveResult{}, err
	}
	if len(c.Items) == 0 {
		return BatchResolveResult{}, fmt.Errorf("%w: items 不能为空", domain.ErrInvalidInput)
	}
	key := idemKey(projectID, "resolve-batch", c.IdempotencyKey)
	var out BatchResolveResult
	err := a.repo.Update(projectID, "findings.resolved", c.Actor, func(s *store.Snapshot) error {
		if old, ok, err := replay[BatchResolveResult](s, key); err != nil {
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
		if p.Status != domain.StatusNeedsFix {
			return domain.ErrInvalidState
		}
		run, ok := s.Validations[p.CurrentRevisionID]
		if !ok || run.RevisionID != p.CurrentRevisionID || run.RuleDigest != domain.RuleDigest(p) || run.GlossaryVersion != p.GlossaryVersion {
			return domain.ErrStaleValidation
		}

		indices := make(map[string]int, len(run.Findings))
		for i := range run.Findings {
			indices[run.Findings[i].ID] = i
		}
		seen := make(map[string]bool, len(c.Items))
		for _, item := range c.Items {
			id := strings.TrimSpace(item.FindingID)
			if id == "" || strings.TrimSpace(item.ResolutionNote) == "" {
				return fmt.Errorf("%w: findingID 和 resolutionNote 不能为空", domain.ErrInvalidInput)
			}
			if seen[id] {
				return fmt.Errorf("%w: findingID %s 重复", domain.ErrInvalidInput, id)
			}
			seen[id] = true
			index, exists := indices[id]
			if !exists {
				return domain.ErrNotFound
			}
			if run.Findings[index].Status != domain.FindingOpen {
				return fmt.Errorf("%w: 问题 %s 已经关闭", domain.ErrInvalidState, id)
			}
		}

		resolved := make([]domain.ValidationFinding, 0, len(c.Items))
		for _, item := range c.Items {
			index := indices[strings.TrimSpace(item.FindingID)]
			run.Findings[index].Status = domain.FindingResolved
			run.Findings[index].ResolutionNote = strings.TrimSpace(item.ResolutionNote)
			resolved = append(resolved, run.Findings[index])
		}
		s.Validations[p.CurrentRevisionID] = run
		updateValidationHistory(s, run)
		p.Touch(a.now())
		s.Projects[projectID] = p
		out = BatchResolveResult{RevisionID: p.CurrentRevisionID, Resolved: domain.SortFindings(resolved), Version: p.Version}
		return remember(s, key, "resolve-batch", out, a.now())
	})
	return out, err
}
