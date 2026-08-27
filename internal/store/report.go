package store

import (
	"sort"
	"stagecaption-finalizer/internal/domain"
	"time"
)

type Statistics struct {
	ProjectCount     int       `json:"projectCount"`
	RevisionCount    int       `json:"revisionCount"`
	FindingCount     int       `json:"findingCount"`
	OpenFindingCount int       `json:"openFindingCount"`
	FrozenCount      int       `json:"frozenCount"`
	LastEventAt      time.Time `json:"lastEventAt"`
}

func (r *Repository) Statistics() Statistics {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var st Statistics
	st.ProjectCount = len(r.state.Projects)
	for _, p := range r.state.Projects {
		if p.Status == domain.StatusFrozen {
			st.FrozenCount++
		}
	}
	for _, items := range r.state.Revisions {
		st.RevisionCount += len(items)
	}
	for _, runs := range r.state.ValidationRuns {
		for _, run := range runs {
			st.FindingCount += len(run.Findings)
			for _, f := range run.Findings {
				if f.Status == domain.FindingOpen {
					st.OpenFindingCount++
				}
			}
		}
	}
	if len(r.events) > 0 {
		st.LastEventAt = r.events[len(r.events)-1].OccurredAt
	}
	return st
}
func (r *Repository) ProjectIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.state.Projects))
	for id := range r.state.Projects {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
func (r *Repository) HasProject(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.state.Projects[id]
	return ok
}
