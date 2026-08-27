package validation

import (
	"fmt"
	"sort"
	"time"

	"stagecaption-finalizer/internal/domain"
)

type Engine struct{ Now func() time.Time }

func New() *Engine { return &Engine{Now: time.Now} }

func (e *Engine) Validate(p domain.CaptionProject, r domain.CaptionRevision, terms []domain.GlossaryTerm) domain.ValidationRun {
	findings := make([]domain.ValidationFinding, 0)
	add := func(code string, severity domain.FindingSeverity, seq int, message, evidence string) {
		key := fmt.Sprintf("%s:%06d:%s", code, seq, evidence)
		findings = append(findings, domain.ValidationFinding{ID: domain.Digest(key)[:20], ProjectID: p.ID, RevisionID: r.ID, RuleCode: code, Severity: severity, CueSequence: seq, Message: message, Evidence: evidence, Status: domain.FindingOpen, CreatedAt: e.Now().UTC()})
	}
	validateTiming(p, r, add)
	validateGlossary(r, terms, add)
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].RuleCode != findings[j].RuleCode {
			return findings[i].RuleCode < findings[j].RuleCode
		}
		if findings[i].CueSequence != findings[j].CueSequence {
			return findings[i].CueSequence < findings[j].CueSequence
		}
		return findings[i].ID < findings[j].ID
	})
	return domain.ValidationRun{RevisionID: r.ID, GlossaryVersion: r.GlossaryVersion, RuleDigest: domain.RuleDigest(p), RuleSummary: p.Rules(), Findings: findings, RanAt: e.Now().UTC()}
}

func HasBlocking(run domain.ValidationRun) bool {
	for _, f := range run.Findings {
		if f.Severity == domain.SeverityBlocking && f.Status == domain.FindingOpen {
			return true
		}
	}
	return false
}
