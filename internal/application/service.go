package application

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"stagecaption-finalizer/internal/domain"
	"stagecaption-finalizer/internal/store"
	"stagecaption-finalizer/internal/validation"
)

type Service struct {
	repo      *store.Repository
	validator *validation.Engine
	now       func() time.Time
}

func New(repo *store.Repository, validator *validation.Engine) *Service {
	return &Service{repo: repo, validator: validator, now: time.Now}
}

func newID(prefix string) string {
	var b [10]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}
func requireKey(key string) error {
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("%w: idempotencyKey 不能为空", domain.ErrInvalidInput)
	}
	return nil
}
func idemKey(project, operation, key string) string { return project + ":" + operation + ":" + key }

func replay[T any](s *store.Snapshot, key string) (T, bool, error) {
	var zero T
	record, ok := s.Idempotency[key]
	if !ok {
		return zero, false, nil
	}
	var out T
	if err := json.Unmarshal(record.Response, &out); err != nil {
		return zero, false, err
	}
	return out, true, nil
}
func remember(s *store.Snapshot, key, operation string, value any, now time.Time) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	s.Idempotency[key] = store.IdempotencyRecord{Operation: operation, Response: b, CreatedAt: now.UTC()}
	return nil
}

func currentTerms(s store.Snapshot, p domain.CaptionProject) []domain.GlossaryTerm {
	return termsAt(s, p.ID, p.GlossaryVersion)
}
func termsAt(s store.Snapshot, projectID string, version int64) []domain.GlossaryTerm {
	out := []domain.GlossaryTerm{}
	for _, t := range s.Terms[projectID] {
		if t.Version == version {
			out = append(out, t)
		}
	}
	return out
}
func findRevision(s store.Snapshot, projectID, revisionID string) (domain.CaptionRevision, error) {
	for _, r := range s.Revisions[projectID] {
		if r.ID == revisionID {
			return r, nil
		}
	}
	return domain.CaptionRevision{}, domain.ErrNotFound
}
func latestReview(s store.Snapshot, projectID, revisionID string) (domain.ReviewDecision, error) {
	items := s.Reviews[projectID]
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].RevisionID == revisionID {
			return items[i], nil
		}
	}
	return domain.ReviewDecision{}, domain.ErrNotFound
}

func (a *Service) GetWorkspace(projectID string) (Workspace, error) {
	var out Workspace
	err := a.repo.Read(func(s store.Snapshot) error {
		p, ok := s.Projects[projectID]
		if !ok {
			return domain.ErrNotFound
		}
		out.Project = p
		out.Terms = currentTerms(s, p)
		out.Revisions = s.Revisions[projectID]
		out.Reviews = s.Reviews[projectID]
		if run, ok := s.Validations[p.CurrentRevisionID]; ok {
			copyRun := run
			out.Validation = &copyRun
		}
		if m, ok := s.Manifests[projectID]; ok {
			copyM := m
			out.Manifest = &copyM
		}
		return nil
	})
	return out, err
}

func (a *Service) ListProjects() ([]domain.CaptionProject, error) {
	out := []domain.CaptionProject{}
	err := a.repo.Read(func(s store.Snapshot) error {
		for _, p := range s.Projects {
			out = append(out, p)
		}
		return nil
	})
	return SortProjects(out), err
}

func IsDomain(err, target error) bool { return errors.Is(err, target) }
