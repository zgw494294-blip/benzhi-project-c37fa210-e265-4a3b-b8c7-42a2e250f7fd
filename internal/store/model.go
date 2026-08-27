package store

import (
	"encoding/json"
	"time"

	"stagecaption-finalizer/internal/domain"
)

type IdempotencyRecord struct {
	Operation string          `json:"operation"`
	Response  json.RawMessage `json:"response"`
	CreatedAt time.Time       `json:"createdAt"`
}

type Snapshot struct {
	Projects       map[string]domain.CaptionProject    `json:"projects"`
	Terms          map[string][]domain.GlossaryTerm    `json:"terms"`
	Revisions      map[string][]domain.CaptionRevision `json:"revisions"`
	Validations    map[string]domain.ValidationRun     `json:"validations"`
	ValidationRuns map[string][]domain.ValidationRun   `json:"validationRuns"`
	Reviews        map[string][]domain.ReviewDecision  `json:"reviews"`
	Manifests      map[string]domain.FreezeManifest    `json:"manifests"`
	Idempotency    map[string]IdempotencyRecord        `json:"idempotency"`
}

func emptySnapshot() Snapshot {
	return Snapshot{Projects: map[string]domain.CaptionProject{}, Terms: map[string][]domain.GlossaryTerm{}, Revisions: map[string][]domain.CaptionRevision{}, Validations: map[string]domain.ValidationRun{}, ValidationRuns: map[string][]domain.ValidationRun{}, Reviews: map[string][]domain.ReviewDecision{}, Manifests: map[string]domain.FreezeManifest{}, Idempotency: map[string]IdempotencyRecord{}}
}

func ensureSnapshot(s *Snapshot) {
	if s.Projects == nil {
		s.Projects = map[string]domain.CaptionProject{}
	}
	if s.Terms == nil {
		s.Terms = map[string][]domain.GlossaryTerm{}
	}
	if s.Revisions == nil {
		s.Revisions = map[string][]domain.CaptionRevision{}
	}
	if s.Validations == nil {
		s.Validations = map[string]domain.ValidationRun{}
	}
	if s.ValidationRuns == nil {
		s.ValidationRuns = map[string][]domain.ValidationRun{}
	}
	if s.Reviews == nil {
		s.Reviews = map[string][]domain.ReviewDecision{}
	}
	if s.Manifests == nil {
		s.Manifests = map[string]domain.FreezeManifest{}
	}
	if s.Idempotency == nil {
		s.Idempotency = map[string]IdempotencyRecord{}
	}
}

type AuditEvent struct {
	Sequence   int64           `json:"sequence"`
	ProjectID  string          `json:"projectID"`
	Type       string          `json:"type"`
	Actor      string          `json:"actor"`
	OccurredAt time.Time       `json:"occurredAt"`
	Previous   string          `json:"previousDigest"`
	Digest     string          `json:"digest"`
	Projection json.RawMessage `json:"projection"`
}

type EventView struct {
	Sequence   int64     `json:"sequence"`
	ProjectID  string    `json:"projectID"`
	Type       string    `json:"type"`
	Actor      string    `json:"actor"`
	OccurredAt time.Time `json:"occurredAt"`
	Previous   string    `json:"previousDigest"`
	Digest     string    `json:"digest"`
}
