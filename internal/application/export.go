package application

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"stagecaption-finalizer/internal/domain"
	"stagecaption-finalizer/internal/store"
)

func srtTime(ms int64) string {
	h := ms / 3600000
	ms %= 3600000
	m := ms / 60000
	ms %= 60000
	s := ms / 1000
	ms %= 1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}

func renderCaptions(cues []domain.CaptionCue) string {
	var b strings.Builder
	for _, c := range cues {
		fmt.Fprintf(&b, "%d\n%s --> %s\n%s\n\n", c.Sequence, srtTime(c.InMillis), srtTime(c.OutMillis), c.TranslatedText)
	}
	return b.String()
}

func (a *Service) Export(projectID string) (ExportBundle, error) {
	var out ExportBundle
	err := a.repo.Read(func(s store.Snapshot) error {
		m, ok := s.Manifests[projectID]
		if !ok {
			return domain.ErrInvalidState
		}
		p := s.Projects[projectID]
		if p.Status != domain.StatusFrozen {
			return domain.ErrInvalidState
		}
		r, err := findRevision(s, projectID, m.RevisionID)
		if err != nil {
			return err
		}
		out = ExportBundle{ProjectID: projectID, VerificationCode: m.VerificationCode, Captions: renderCaptions(r.Cues), Glossary: termsAt(s, projectID, r.GlossaryVersion), Audit: a.repo.Events(projectID), Manifest: m}
		return nil
	})
	return out, err
}

func (a *Service) ExportItem(projectID, kind string) (ExportItem, error) {
	var out ExportItem
	events := a.repo.Events(projectID)
	err := a.repo.Read(func(s store.Snapshot) error {
		manifest, ok := s.Manifests[projectID]
		if !ok || s.Projects[projectID].Status != domain.StatusFrozen {
			return domain.ErrInvalidState
		}
		revision, err := findRevision(s, projectID, manifest.RevisionID)
		if err != nil {
			return err
		}
		var content []byte
		contentType := "application/json; charset=utf-8"
		extension := "json"
		switch kind {
		case "captions":
			content = []byte(renderCaptions(revision.Cues))
			contentType, extension = "application/x-subrip; charset=utf-8", "srt"
		case "glossary":
			content, err = json.MarshalIndent(SortTerms(termsAt(s, projectID, revision.GlossaryVersion)), "", "  ")
			if err == nil {
				content = append(content, '\n')
			}
		case "audit":
			content, err = json.MarshalIndent(events, "", "  ")
			if err == nil {
				content = append(content, '\n')
			}
		default:
			return fmt.Errorf("%w: 导出分项必须是 captions、glossary 或 audit", domain.ErrInvalidInput)
		}
		if err != nil {
			return err
		}
		out = ExportItem{ProjectID: projectID, ManifestID: manifest.ID, VerificationCode: manifest.VerificationCode, Kind: kind, ContentType: contentType, Filename: fmt.Sprintf("%s-%s.%s", projectID, kind, extension), Digest: domain.DigestBytes(content), Content: content}
		return nil
	})
	return out, err
}

func (a *Service) Verify(projectID string) (VerificationResult, error) {
	result := VerificationResult{Checks: map[string]bool{}}
	auditAnchor := ""
	err := a.repo.Read(func(s store.Snapshot) error {
		m, ok := s.Manifests[projectID]
		if !ok {
			return domain.ErrNotFound
		}
		p, ok := s.Projects[projectID]
		if !ok {
			return domain.ErrNotFound
		}
		r, err := findRevision(s, projectID, m.RevisionID)
		if err != nil {
			return err
		}
		d, err := latestReview(s, projectID, m.RevisionID)
		if err != nil {
			return err
		}
		terms := termsAt(s, projectID, r.GlossaryVersion)
		result.Checks["manifestCode"] = m.Verify()
		result.Checks["projectFrozen"] = p.Status == domain.StatusFrozen
		result.Checks["captionDigest"] = domain.RevisionDigest(r.Cues) == m.CaptionDigest
		result.Checks["glossaryDigest"] = domain.GlossaryDigest(terms) == m.GlossaryDigest
		result.Checks["ruleDigest"] = domain.RuleDigest(p) == m.RuleDigest
		result.Checks["approvedReview"] = d.ID == m.ReviewDecisionID && d.Decision == "approve"
		auditAnchor = m.AuditHeadDigest
		result.VerificationCode = m.VerificationCode
		return nil
	})
	if err != nil {
		return result, err
	}
	result.Checks["auditChain"] = a.repo.VerifyAudit() == nil
	result.Checks["auditAnchor"] = a.repo.VerifyAuditAnchor(auditAnchor)
	result.Checks["projection"] = a.repo.VerifyProjection() == nil
	result.Valid = true
	for _, ok := range result.Checks {
		if !ok {
			result.Valid = false
		}
	}
	if result.Valid {
		result.Message = "冻结清单、字幕、术语、规则、复核决定和审计摘要链均完整"
	} else {
		result.Message = "完整性核验失败，冻结内容或历史事件可能已被篡改"
	}
	return result, err
}

func ExportFilename(projectID, kind string) string {
	return fmt.Sprintf("%s-%s-%s", projectID, time.Now().UTC().Format("20060102"), kind)
}
