package forbidden_scope_test

import (
	"testing"

	"stagecaption-finalizer/internal/application"
	"stagecaption-finalizer/internal/domain"
	"stagecaption-finalizer/internal/store"
	"stagecaption-finalizer/internal/validation"
)

func TestForbiddenTranslationOnlyAppliesToMatchingSource(t *testing.T) {
	repo, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc := application.New(repo, validation.New())
	p, err := svc.CreateProject(application.CreateProjectCommand{Title: "演出", SourceLanguage: "en", TargetLanguage: "zh", FrameRate: 25, MinDisplayMillis: 500, MaxDisplayMillis: 5000, Actor: "负责人", IdempotencyKey: "create"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.AddTerm(p.ID, application.TermCommand{SourceText: "King", RequiredTranslation: "王", ForbiddenTranslations: []string{"国王"}, ExpectedVersion: p.Version, Actor: "译员", IdempotencyKey: "term"}); err != nil {
		t.Fatal(err)
	}
	w, err := svc.GetWorkspace(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.SubmitRevision(p.ID, application.RevisionCommand{SubmittedBy: "译员", ExpectedVersion: w.Project.Version, IdempotencyKey: "revision", Cues: []domain.CaptionCue{{Sequence: 1, InMillis: 0, OutMillis: 1200, SourceText: "Queen", TranslatedText: "国王"}}}); err != nil {
		t.Fatal(err)
	}
	w, err = svc.GetWorkspace(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	run, err := svc.Validate(p.ID, application.VersionedCommand{ExpectedVersion: w.Project.Version, Actor: "译员", IdempotencyKey: "validate"})
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Findings) != 0 {
		t.Fatalf("unexpected forbidden finding: got %d findings", len(run.Findings))
	}
}
