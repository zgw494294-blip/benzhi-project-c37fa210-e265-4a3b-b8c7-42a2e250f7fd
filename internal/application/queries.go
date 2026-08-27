package application

import (
	"sort"
	"stagecaption-finalizer/internal/domain"
	"stagecaption-finalizer/internal/store"
)

func (a *Service) Stats() store.Statistics { return a.repo.Statistics() }

func (a *Service) Findings(projectID string) ([]domain.ValidationFinding, error) {
	var out []domain.ValidationFinding
	err := a.repo.Read(func(s store.Snapshot) error {
		p, ok := s.Projects[projectID]
		if !ok {
			return domain.ErrNotFound
		}
		run, ok := s.Validations[p.CurrentRevisionID]
		if !ok {
			return nil
		}
		out = domain.SortFindings(run.Findings)
		return nil
	})
	return out, err
}

func (a *Service) Audit(projectID string) []map[string]any {
	events := a.repo.Events(projectID)
	out := make([]map[string]any, 0, len(events))
	for _, e := range events {
		out = append(out, map[string]any{"sequence": e.Sequence, "type": e.Type, "actor": e.Actor, "occurredAt": e.OccurredAt, "digest": e.Digest})
	}
	return out
}
func SortProjects(projects []domain.CaptionProject) []domain.CaptionProject {
	out := append(make([]domain.CaptionProject, 0, len(projects)), projects...)
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.Before(out[j].UpdatedAt) })
	return out
}
