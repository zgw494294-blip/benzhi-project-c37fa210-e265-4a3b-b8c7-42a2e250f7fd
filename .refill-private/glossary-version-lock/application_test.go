package glossary_version_lock_test

import (
	"testing"

	"stagecaption-finalizer/internal/application"
	"stagecaption-finalizer/internal/domain"
	"stagecaption-finalizer/internal/store"
	"stagecaption-finalizer/internal/validation"
)

func TestGlossaryChangeAllowsReplacementRevisionAfterValidation(t *testing.T) {
	repo, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc := application.New(repo, validation.New())
	p, err := svc.CreateProject(application.CreateProjectCommand{Title: "演出", SourceLanguage: "en", TargetLanguage: "zh", FrameRate: 25, MinDisplayMillis: 500, MaxDisplayMillis: 5000, Actor: "负责人", IdempotencyKey: "create"})
	if err != nil {
		t.Fatal(err)
	}
	r, err := svc.SubmitRevision(p.ID, application.RevisionCommand{SubmittedBy: "译员", ExpectedVersion: p.Version, IdempotencyKey: "revision-1", Cues: []domain.CaptionCue{{Sequence: 1, InMillis: 0, OutMillis: 1200, SourceText: "Hello", TranslatedText: "你好"}}})
	if err != nil {
		t.Fatal(err)
	}
	w, err := svc.GetWorkspace(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.Validate(p.ID, application.VersionedCommand{ExpectedVersion: w.Project.Version, Actor: "译员", IdempotencyKey: "validate"}); err != nil {
		t.Fatal(err)
	}
	w, err = svc.GetWorkspace(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.AddTerm(p.ID, application.TermCommand{SourceText: "World", RequiredTranslation: "世界", ExpectedVersion: w.Project.Version, Actor: "译员", IdempotencyKey: "term-after-validation"}); err != nil {
		t.Fatal(err)
	}
	w, err = svc.GetWorkspace(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.SubmitRevision(p.ID, application.RevisionCommand{ParentRevisionID: r.ID, SubmittedBy: "译员", ExpectedVersion: w.Project.Version, IdempotencyKey: "revision-2", Cues: []domain.CaptionCue{{Sequence: 1, InMillis: 0, OutMillis: 1200, SourceText: "Hello", TranslatedText: "你好"}}})
	if err != nil {
		t.Fatalf("replacement revision blocked after glossary change: %v", err)
	}
}
