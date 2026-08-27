package domain

import "time"

func NewManifest(id string, p CaptionProject, r CaptionRevision, terms []GlossaryTerm, decision ReviewDecision, auditHead, actor string, now time.Time) FreezeManifest {
	m := FreezeManifest{ID: id, ProjectID: p.ID, RevisionID: r.ID, ReviewDecisionID: decision.ID, RuleDigest: RuleDigest(p), GlossaryDigest: GlossaryDigest(terms), CaptionDigest: r.ContentDigest, AuditHeadDigest: auditHead, FrozenBy: actor, FrozenAt: now.UTC()}
	m.VerificationCode = ManifestCode(m)
	return m
}

func (m FreezeManifest) Verify() bool { return m.VerificationCode == ManifestCode(m) }
