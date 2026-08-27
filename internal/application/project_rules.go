package application

import (
	"stagecaption-finalizer/internal/domain"
	"stagecaption-finalizer/internal/store"
)

func (a *Service) UpdateProjectRules(projectID string, c RuleUpdateCommand) (domain.CaptionProject, error) {
	if err := requireKey(c.IdempotencyKey); err != nil {
		return domain.CaptionProject{}, err
	}
	key := idemKey(projectID, "rules", c.IdempotencyKey)
	var out domain.CaptionProject
	err := a.repo.Update(projectID, "project.rules-updated", c.Actor, func(s *store.Snapshot) error {
		if old, ok, err := replay[domain.CaptionProject](s, key); err != nil {
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
		if err := p.UpdateRules(c.Title, c.SourceLanguage, c.TargetLanguage, c.FrameRate, c.MinDisplayMillis, c.MaxDisplayMillis, a.now()); err != nil {
			return err
		}
		if p.CurrentRevisionID != "" {
			delete(s.Validations, p.CurrentRevisionID)
		}
		s.Projects[projectID] = p
		out = p
		return remember(s, key, "rules", out, a.now())
	})
	return out, err
}
