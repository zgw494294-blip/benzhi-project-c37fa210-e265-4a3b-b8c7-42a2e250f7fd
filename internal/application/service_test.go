package application

import (
	"stagecaption-finalizer/internal/domain"
	"stagecaption-finalizer/internal/store"
	"stagecaption-finalizer/internal/validation"
	"testing"
)

func TestCompleteFlowAndGuards(t *testing.T) {
	repo, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc := New(repo, validation.New())
	p, err := svc.CreateProject(CreateProjectCommand{Title: "哈姆雷特", SourceLanguage: "en", TargetLanguage: "zh", FrameRate: 25, MinDisplayMillis: 500, MaxDisplayMillis: 5000, Actor: "负责人", IdempotencyKey: "c1"})
	if err != nil {
		t.Fatal(err)
	}
	term, err := svc.AddTerm(p.ID, TermCommand{SourceText: "King", RequiredTranslation: "王", ForbiddenTranslations: []string{"国王"}, ExpectedVersion: p.Version, Actor: "译员", IdempotencyKey: "t1"})
	if err != nil {
		t.Fatal(err)
	}
	if term.ID == "" {
		t.Fatal("术语未创建")
	}
	w, _ := svc.GetWorkspace(p.ID)
	r, err := svc.SubmitRevision(p.ID, RevisionCommand{SubmittedBy: "译员", ExpectedVersion: w.Project.Version, IdempotencyKey: "r1", Cues: []domain.CaptionCue{{Sequence: 1, InMillis: 0, OutMillis: 1200, SourceText: "King", TranslatedText: "王"}}})
	if err != nil {
		t.Fatal(err)
	}
	w, _ = svc.GetWorkspace(p.ID)
	run, err := svc.Validate(p.ID, VersionedCommand{ExpectedVersion: w.Project.Version, Actor: "译员", IdempotencyKey: "v1"})
	if err != nil || len(run.Findings) != 0 {
		t.Fatalf("校验失败: %v %#v", err, run.Findings)
	}
	w, _ = svc.GetWorkspace(p.ID)
	_, err = svc.SubmitForReview(p.ID, VersionedCommand{ExpectedVersion: w.Project.Version, Actor: "译员", IdempotencyKey: "s1"})
	if err != nil {
		t.Fatal(err)
	}
	w, _ = svc.GetWorkspace(p.ID)
	_, err = svc.Review(p.ID, ReviewCommand{Decision: "approve", Reviewer: r.SubmittedBy, ExpectedVersion: w.Project.Version, IdempotencyKey: "bad"})
	if err == nil {
		t.Fatal("提交者不应能复核")
	}
	_, err = svc.Review(p.ID, ReviewCommand{Decision: "approve", Reviewer: "校审", ExpectedVersion: w.Project.Version, IdempotencyKey: "ok"})
	if err != nil {
		t.Fatal(err)
	}
	w, _ = svc.GetWorkspace(p.ID)
	_, err = svc.Freeze(p.ID, VersionedCommand{ExpectedVersion: w.Project.Version, Actor: "负责人", IdempotencyKey: "f1"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := svc.Verify(p.ID)
	if err != nil || !result.Valid {
		t.Fatalf("核验失败: %v %#v", err, result)
	}
}
