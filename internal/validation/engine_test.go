package validation

import (
	"stagecaption-finalizer/internal/domain"
	"testing"
	"time"
)

func TestStableFindings(t *testing.T) {
	p, _ := domain.NewProject("p", "戏剧", "en", "zh", 25, 500, 5000, time.Now())
	r := domain.CaptionRevision{ID: "r", ProjectID: "p", GlossaryVersion: 1, Cues: []domain.CaptionCue{{Sequence: 2, InMillis: 0, OutMillis: 100, SourceText: "King", TranslatedText: "国王"}}}
	term, _ := domain.NewTerm("t", "p", "King", "王", []string{"国王"}, false, 1, time.Now())
	e := New()
	now := time.Unix(1, 0)
	e.Now = func() time.Time { return now }
	a := e.Validate(*p, r, []domain.GlossaryTerm{term})
	b := e.Validate(*p, r, []domain.GlossaryTerm{term})
	if domain.Digest(a.Findings) != domain.Digest(b.Findings) || len(a.Findings) < 3 {
		t.Fatalf("校验不稳定或规则缺失: %#v", a.Findings)
	}
}
