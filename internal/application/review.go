package application

import (
	"fmt"
	"stagecaption-finalizer/internal/domain"
	"stagecaption-finalizer/internal/store"
	"stagecaption-finalizer/internal/validation"
	"strings"
)

func (a *Service) SubmitForReview(projectID string, c VersionedCommand) (domain.CaptionProject, error) {
	if err := requireKey(c.IdempotencyKey); err != nil {
		return domain.CaptionProject{}, err
	}
	key := idemKey(projectID, "submit-review", c.IdempotencyKey)
	var out domain.CaptionProject
	err := a.repo.Update(projectID, "review.submitted", c.Actor, func(s *store.Snapshot) error {
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
		run, ok := s.Validations[p.CurrentRevisionID]
		if !ok || run.RevisionID != p.CurrentRevisionID || run.GlossaryVersion != p.GlossaryVersion || run.RuleDigest != domain.RuleDigest(p) {
			return domain.ErrStaleValidation
		}
		if validation.HasBlocking(run) {
			return domain.ErrBlockingFindings
		}
		if err := p.SubmitReview(a.now()); err != nil {
			return err
		}
		s.Projects[projectID] = p
		out = p
		return remember(s, key, "submit-review", out, a.now())
	})
	return out, err
}

func (a *Service) Review(projectID string, c ReviewCommand) (domain.ReviewDecision, error) {
	if err := requireKey(c.IdempotencyKey); err != nil {
		return domain.ReviewDecision{}, err
	}
	decision := strings.ToLower(strings.TrimSpace(c.Decision))
	reviewer := strings.TrimSpace(c.Reviewer)
	if decision != "approve" && decision != "return" {
		return domain.ReviewDecision{}, fmt.Errorf("%w: decision 必须是 approve 或 return", domain.ErrInvalidInput)
	}
	if decision == "return" && strings.TrimSpace(c.Reason) == "" {
		return domain.ReviewDecision{}, fmt.Errorf("%w: 退回必须填写理由", domain.ErrInvalidInput)
	}
	key := idemKey(projectID, "review", c.IdempotencyKey)
	var out domain.ReviewDecision
	err := a.repo.Update(projectID, "review."+decision, reviewer, func(s *store.Snapshot) error {
		if old, ok, err := replay[domain.ReviewDecision](s, key); err != nil {
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
		r, err := findRevision(*s, projectID, p.CurrentRevisionID)
		if err != nil {
			return err
		}
		if reviewer == "" || reviewer == strings.TrimSpace(r.SubmittedBy) {
			return domain.ErrIdentityConflict
		}
		approved := decision == "approve"
		if err = p.ApplyReview(approved, a.now()); err != nil {
			return err
		}
		d := domain.ReviewDecision{ID: newID("review"), ProjectID: projectID, RevisionID: r.ID, Reviewer: reviewer, Decision: decision, Reason: strings.TrimSpace(c.Reason), DecidedAt: a.now().UTC()}
		s.Reviews[projectID] = append(s.Reviews[projectID], d)
		s.Projects[projectID] = p
		out = d
		return remember(s, key, "review", out, a.now())
	})
	return out, err
}

func (a *Service) Freeze(projectID string, c VersionedCommand) (domain.FreezeManifest, error) {
	if err := requireKey(c.IdempotencyKey); err != nil {
		return domain.FreezeManifest{}, err
	}
	key := idemKey(projectID, "freeze", c.IdempotencyKey)
	var out domain.FreezeManifest
	auditHead := a.repo.AuditHead()
	err := a.repo.Update(projectID, "project.frozen", c.Actor, func(s *store.Snapshot) error {
		if old, ok, err := replay[domain.FreezeManifest](s, key); err != nil {
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
		r, err := findRevision(*s, projectID, p.CurrentRevisionID)
		if err != nil {
			return err
		}
		d, err := latestReview(*s, projectID, r.ID)
		if err != nil || d.Decision != "approve" {
			return domain.ErrInvalidState
		}
		run, ok := s.Validations[r.ID]
		if !ok || run.RuleDigest != domain.RuleDigest(p) || run.GlossaryVersion != r.GlossaryVersion || validation.HasBlocking(run) {
			return domain.ErrStaleValidation
		}
		if err = p.Freeze(a.now()); err != nil {
			return err
		}
		m := domain.NewManifest(newID("manifest"), p, r, termsAt(*s, projectID, r.GlossaryVersion), d, auditHead, c.Actor, a.now())
		s.Manifests[projectID] = m
		s.Projects[projectID] = p
		out = m
		return remember(s, key, "freeze", out, a.now())
	})
	return out, err
}
