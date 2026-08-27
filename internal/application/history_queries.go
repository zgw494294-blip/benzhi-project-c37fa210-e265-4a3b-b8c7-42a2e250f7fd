package application

import (
	"fmt"
	"sort"
	"strings"

	"stagecaption-finalizer/internal/domain"
	"stagecaption-finalizer/internal/store"
)

type FindingFilter struct {
	Severity    string
	RuleCode    string
	Status      string
	CueSequence *int
}

func (a *Service) ListRevisions(projectID string) ([]domain.CaptionRevision, error) {
	out := []domain.CaptionRevision{}
	err := a.repo.Read(func(s store.Snapshot) error {
		if _, ok := s.Projects[projectID]; !ok {
			return domain.ErrNotFound
		}
		out = append(out, s.Revisions[projectID]...)
		sort.SliceStable(out, func(i, j int) bool {
			if out[i].RevisionNumber != out[j].RevisionNumber {
				return out[i].RevisionNumber < out[j].RevisionNumber
			}
			return out[i].ID < out[j].ID
		})
		return nil
	})
	return out, err
}

func (a *Service) GetRevision(projectID, revisionID string) (domain.CaptionRevision, error) {
	var out domain.CaptionRevision
	err := a.repo.Read(func(s store.Snapshot) error {
		if _, ok := s.Projects[projectID]; !ok {
			return domain.ErrNotFound
		}
		var err error
		out, err = findRevision(s, projectID, revisionID)
		return err
	})
	return out, err
}

func (a *Service) DiffRevisions(projectID, fromID, toID string) (RevisionDiff, error) {
	if strings.TrimSpace(fromID) == "" || strings.TrimSpace(toID) == "" || fromID == toID {
		return RevisionDiff{}, fmt.Errorf("%w: from 和 to 必须是两个不同修订", domain.ErrInvalidInput)
	}
	var out RevisionDiff
	err := a.repo.Read(func(s store.Snapshot) error {
		if _, ok := s.Projects[projectID]; !ok {
			return domain.ErrNotFound
		}
		from, err := findRevision(s, projectID, fromID)
		if err != nil {
			return domain.ErrNotFound
		}
		to, err := findRevision(s, projectID, toID)
		if err != nil {
			return domain.ErrNotFound
		}
		out = RevisionDiff{ProjectID: projectID, From: from, To: to, Changes: domain.DiffCues(from.Cues, to.Cues)}
		return nil
	})
	return out, err
}

func validateFindingFilter(filter FindingFilter) error {
	if filter.Severity != "" && filter.Severity != string(domain.SeverityBlocking) && filter.Severity != string(domain.SeverityWarning) {
		return fmt.Errorf("%w: severity 无效", domain.ErrInvalidInput)
	}
	if filter.Status != "" && filter.Status != string(domain.FindingOpen) && filter.Status != string(domain.FindingResolved) {
		return fmt.Errorf("%w: FindingStatus 无效", domain.ErrInvalidInput)
	}
	if filter.CueSequence != nil && *filter.CueSequence < 0 {
		return fmt.Errorf("%w: cueSequence 无效", domain.ErrInvalidInput)
	}
	return nil
}

func (a *Service) FilterFindings(projectID string, filter FindingFilter) ([]domain.ValidationFinding, error) {
	if err := validateFindingFilter(filter); err != nil {
		return nil, err
	}
	var out []domain.ValidationFinding
	err := a.repo.Read(func(s store.Snapshot) error {
		p, ok := s.Projects[projectID]
		if !ok {
			return domain.ErrNotFound
		}
		run, ok := s.Validations[p.CurrentRevisionID]
		if !ok {
			out = []domain.ValidationFinding{}
			return nil
		}
		for _, finding := range run.Findings {
			if filter.Severity != "" && string(finding.Severity) != filter.Severity {
				continue
			}
			if filter.RuleCode != "" && finding.RuleCode != filter.RuleCode {
				continue
			}
			if filter.Status != "" && string(finding.Status) != filter.Status {
				continue
			}
			if filter.CueSequence != nil && finding.CueSequence != *filter.CueSequence {
				continue
			}
			out = append(out, finding)
		}
		out = domain.SortFindings(out)
		return nil
	})
	return out, err
}

func validationRunsFor(s store.Snapshot, revisionID string) []domain.ValidationRun {
	runs := append([]domain.ValidationRun(nil), s.ValidationRuns[revisionID]...)
	if len(runs) == 0 {
		if run, ok := s.Validations[revisionID]; ok {
			runs = append(runs, run)
		}
	}
	sort.SliceStable(runs, func(i, j int) bool {
		if !runs[i].RanAt.Equal(runs[j].RanAt) {
			return runs[i].RanAt.Before(runs[j].RanAt)
		}
		return runs[i].ID < runs[j].ID
	})
	return runs
}

func runSummary(run domain.ValidationRun) ValidationRunSummary {
	return ValidationRunSummary{ID: run.ID, RevisionID: run.RevisionID, RanAt: run.RanAt, RuleDigest: run.RuleDigest, RuleSummary: run.RuleSummary, GlossaryVersion: run.GlossaryVersion, Counts: SummarizeFindings(run.Findings)}
}

func (a *Service) ValidationRuns(projectID, revisionID string) ([]ValidationRunSummary, error) {
	out := []ValidationRunSummary{}
	err := a.repo.Read(func(s store.Snapshot) error {
		if _, ok := s.Projects[projectID]; !ok {
			return domain.ErrNotFound
		}
		if _, err := findRevision(s, projectID, revisionID); err != nil {
			return domain.ErrNotFound
		}
		for _, run := range validationRunsFor(s, revisionID) {
			out = append(out, runSummary(run))
		}
		return nil
	})
	return out, err
}

func findRunForProject(s store.Snapshot, projectID, runID string) (domain.ValidationRun, error) {
	for _, revision := range s.Revisions[projectID] {
		for _, run := range validationRunsFor(s, revision.ID) {
			if run.ID == runID {
				return run, nil
			}
		}
	}
	return domain.ValidationRun{}, domain.ErrNotFound
}

func (a *Service) CompareValidationRuns(projectID, beforeID, afterID string) (ValidationComparison, error) {
	if beforeID == "" || afterID == "" || beforeID == afterID {
		return ValidationComparison{}, fmt.Errorf("%w: before 和 after 必须是两个不同运行", domain.ErrInvalidInput)
	}
	var out ValidationComparison
	err := a.repo.Read(func(s store.Snapshot) error {
		if _, ok := s.Projects[projectID]; !ok {
			return domain.ErrNotFound
		}
		before, err := findRunForProject(s, projectID, beforeID)
		if err != nil {
			return err
		}
		after, err := findRunForProject(s, projectID, afterID)
		if err != nil {
			return err
		}
		left, right := map[string]domain.ValidationFinding{}, map[string]domain.ValidationFinding{}
		for _, finding := range before.Findings {
			left[finding.ID] = finding
		}
		for _, finding := range after.Findings {
			right[finding.ID] = finding
		}
		out = ValidationComparison{BeforeID: beforeID, AfterID: afterID, Added: []domain.ValidationFinding{}, Disappeared: []domain.ValidationFinding{}, Persistent: []FindingTransition{}}
		for id, finding := range right {
			if old, ok := left[id]; ok {
				out.Persistent = append(out.Persistent, FindingTransition{Before: old, After: finding})
			} else {
				out.Added = append(out.Added, finding)
			}
		}
		for id, finding := range left {
			if _, ok := right[id]; !ok {
				out.Disappeared = append(out.Disappeared, finding)
			}
		}
		out.Added = domain.SortFindings(out.Added)
		out.Disappeared = domain.SortFindings(out.Disappeared)
		sort.SliceStable(out.Persistent, func(i, j int) bool {
			a, b := out.Persistent[i].After, out.Persistent[j].After
			if a.RuleCode != b.RuleCode {
				return a.RuleCode < b.RuleCode
			}
			if a.CueSequence != b.CueSequence {
				return a.CueSequence < b.CueSequence
			}
			return a.ID < b.ID
		})
		return nil
	})
	return out, err
}

func (a *Service) Glossary(projectID string, version int64) ([]domain.GlossaryTerm, error) {
	out := []domain.GlossaryTerm{}
	err := a.repo.Read(func(s store.Snapshot) error {
		p, ok := s.Projects[projectID]
		if !ok {
			return domain.ErrNotFound
		}
		if version < 0 || version > p.GlossaryVersion {
			return domain.ErrNotFound
		}
		out = SortTerms(termsAt(s, projectID, version))
		return nil
	})
	return out, err
}
