package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

func Digest(value any) string {
	b, _ := json.Marshal(value)
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func DigestBytes(value []byte) string {
	s := sha256.Sum256(value)
	return hex.EncodeToString(s[:])
}

func RevisionDigest(cues []CaptionCue) string { return Digest(cues) }

func GlossaryDigest(terms []GlossaryTerm) string {
	copyTerms := append([]GlossaryTerm(nil), terms...)
	sort.Slice(copyTerms, func(i, j int) bool { return copyTerms[i].ID < copyTerms[j].ID })
	return Digest(copyTerms)
}

func RuleDigest(p CaptionProject) string {
	return Digest(struct {
		Title, SourceLanguage, TargetLanguage string
		FrameRate                             float64
		Min, Max                              int64
	}{p.Title, p.SourceLanguage, p.TargetLanguage, p.FrameRate, p.MinDisplayMillis, p.MaxDisplayMillis})
}

func ManifestCode(m FreezeManifest) string {
	return Digest(struct{ Project, Revision, Review, Rule, Glossary, Caption, Audit string }{m.ProjectID, m.RevisionID, m.ReviewDecisionID, m.RuleDigest, m.GlossaryDigest, m.CaptionDigest, m.AuditHeadDigest})
}
