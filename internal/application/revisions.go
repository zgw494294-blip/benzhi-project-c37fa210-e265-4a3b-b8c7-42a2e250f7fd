package application

import (
	"fmt"
	"strings"

	"stagecaption-finalizer/internal/domain"
	"stagecaption-finalizer/internal/store"
	"stagecaption-finalizer/internal/validation"
)

func (a *Service) SubmitRevision(projectID string, c RevisionCommand) (domain.CaptionRevision, error) {
	if err := requireKey(c.IdempotencyKey); err != nil {
		return domain.CaptionRevision{}, err
	}
	if strings.TrimSpace(c.SubmittedBy) == "" {
		return domain.CaptionRevision{}, fmt.Errorf("%w: submittedBy 不能为空", domain.ErrInvalidInput)
	}
	key := idemKey(projectID, "revision", c.IdempotencyKey)
	var out domain.CaptionRevision
	err := a.repo.Update(projectID, "revision.submitted", c.SubmittedBy, func(s *store.Snapshot) error {
		if old, ok, err := replay[domain.CaptionRevision](s, key); err != nil {
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
		if !p.CanSubmitRevision() {
			return domain.ErrInvalidState
		}
		parentID := c.ParentRevisionID
		if parentID == "" && p.CurrentRevisionID != "" {
			parentID = p.CurrentRevisionID
		}
		if parentID != "" && parentID != p.CurrentRevisionID {
			return fmt.Errorf("%w: parentRevisionID 不是当前修订", domain.ErrInvalidInput)
		}
		summary := strings.TrimSpace(c.Summary)
		if summary == "" {
			summary = fmt.Sprintf("共 %d 条字幕", len(c.Cues))
		}
		r := domain.CaptionRevision{ID: newID("revision"), ProjectID: projectID, ParentRevisionID: parentID, RevisionNumber: len(s.Revisions[projectID]) + 1, SubmittedBy: c.SubmittedBy, SubmittedAt: a.now().UTC(), GlossaryVersion: p.GlossaryVersion, Summary: summary, Cues: append([]domain.CaptionCue(nil), c.Cues...)}
		r.ContentDigest = domain.RevisionDigest(r.Cues)
		if err := p.SetRevision(r.ID, parentID != "", a.now()); err != nil {
			return err
		}
		s.Revisions[projectID] = append(s.Revisions[projectID], r)
		s.Projects[projectID] = p
		out = r
		return remember(s, key, "revision", out, a.now())
	})
	return out, err
}

func (a *Service) Validate(projectID string, c VersionedCommand) (domain.ValidationRun, error) {
	if err := requireKey(c.IdempotencyKey); err != nil {
		return domain.ValidationRun{}, err
	}
	key := idemKey(projectID, "validate", c.IdempotencyKey)
	var out domain.ValidationRun
	err := a.repo.Update(projectID, "revision.validated", c.Actor, func(s *store.Snapshot) error {
		if old, ok, err := replay[domain.ValidationRun](s, key); err != nil {
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
		terms := termsAt(*s, projectID, r.GlossaryVersion)
		run := a.validator.Validate(p, r, terms)
		run.ID = newID("validation")
		if err = p.MarkValidated(validation.HasBlocking(run), a.now()); err != nil {
			return err
		}
		s.Validations[r.ID] = run
		s.ValidationRuns[r.ID] = append(s.ValidationRuns[r.ID], run)
		s.Projects[projectID] = p
		out = run
		return remember(s, key, "validate", out, a.now())
	})
	return out, err
}

func (a *Service) ResolveFinding(projectID string, c ResolveCommand) (domain.ValidationFinding, error) {
	if err := requireKey(c.IdempotencyKey); err != nil {
		return domain.ValidationFinding{}, err
	}
	if strings.TrimSpace(c.ResolutionNote) == "" {
		return domain.ValidationFinding{}, fmt.Errorf("%w: 整改说明不能为空", domain.ErrInvalidInput)
	}
	key := idemKey(projectID, "resolve", c.IdempotencyKey)
	var out domain.ValidationFinding
	err := a.repo.Update(projectID, "finding.resolved", c.Actor, func(s *store.Snapshot) error {
		if old, ok, err := replay[domain.ValidationFinding](s, key); err != nil {
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
		if !ok {
			return domain.ErrStaleValidation
		}
		if run.RuleDigest != domain.RuleDigest(p) || run.GlossaryVersion != p.GlossaryVersion {
			return domain.ErrStaleValidation
		}
		found := false
		for i := range run.Findings {
			if run.Findings[i].ID == c.FindingID {
				if run.Findings[i].Status != domain.FindingOpen {
					return fmt.Errorf("%w: 问题已经关闭", domain.ErrInvalidState)
				}
				run.Findings[i].ResolutionNote = c.ResolutionNote
				run.Findings[i].Status = domain.FindingResolved
				out = run.Findings[i]
				found = true
				break
			}
		}
		if !found {
			return domain.ErrNotFound
		}
		s.Validations[p.CurrentRevisionID] = run
		updateValidationHistory(s, run)
		p.Touch(a.now())
		s.Projects[projectID] = p
		return remember(s, key, "resolve", out, a.now())
	})
	return out, err
}

func updateValidationHistory(s *store.Snapshot, run domain.ValidationRun) {
	runs := s.ValidationRuns[run.RevisionID]
	for i := range runs {
		if runs[i].ID == run.ID {
			runs[i] = run
			s.ValidationRuns[run.RevisionID] = runs
			return
		}
	}
}
