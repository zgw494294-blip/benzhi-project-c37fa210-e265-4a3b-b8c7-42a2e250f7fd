package application

import (
	"stagecaption-finalizer/internal/domain"
	"stagecaption-finalizer/internal/store"
)

func (a *Service) ReviewDetail(projectID string) (ReviewDetail, error) {
	var out ReviewDetail
	err := a.repo.Read(func(s store.Snapshot) error {
		p, ok := s.Projects[projectID]
		if !ok {
			return domain.ErrNotFound
		}
		current, err := findRevision(s, projectID, p.CurrentRevisionID)
		if err != nil {
			return err
		}
		out.Project = p
		out.CurrentRevision = current
		out.GlossaryVersion = current.GlossaryVersion
		out.Decisions = append([]domain.ReviewDecision(nil), s.Reviews[projectID]...)
		if current.ParentRevisionID == "" {
			out.InitialRevision = true
		} else if parent, findErr := findRevision(s, projectID, current.ParentRevisionID); findErr == nil {
			out.ParentRevision = &parent
			out.Diff = domain.DiffCues(parent.Cues, current.Cues)
		} else {
			out.InitialRevision = true
		}
		if run, ok := s.Validations[current.ID]; ok {
			summary := runSummary(run)
			out.Validation = &summary
		}
		return nil
	})
	return out, err
}

func (a *Service) FreezePreview(projectID string) (FreezePreview, error) {
	var out FreezePreview
	auditHead := a.repo.AuditHead()
	err := a.repo.Read(func(s store.Snapshot) error {
		p, ok := s.Projects[projectID]
		if !ok {
			return domain.ErrNotFound
		}
		if p.Status != domain.StatusApproved {
			return domain.ErrInvalidState
		}
		revision, err := findRevision(s, projectID, p.CurrentRevisionID)
		if err != nil {
			return err
		}
		decision, err := latestReview(s, projectID, revision.ID)
		if err != nil || decision.Decision != "approve" {
			return domain.ErrInvalidState
		}
		run, ok := s.Validations[revision.ID]
		if !ok || run.RuleDigest != domain.RuleDigest(p) || run.GlossaryVersion != revision.GlossaryVersion || len(domain.OpenBlocking(run.Findings)) > 0 {
			return domain.ErrStaleValidation
		}
		terms := termsAt(s, projectID, revision.GlossaryVersion)
		out = FreezePreview{ProjectID: projectID, Revision: revision, Rules: p.Rules(), RuleDigest: domain.RuleDigest(p), GlossaryVersion: revision.GlossaryVersion, Glossary: SortTerms(terms), GlossaryDigest: domain.GlossaryDigest(terms), CaptionDigest: revision.ContentDigest, Review: decision, AuditHeadDigest: auditHead}
		return nil
	})
	return out, err
}
