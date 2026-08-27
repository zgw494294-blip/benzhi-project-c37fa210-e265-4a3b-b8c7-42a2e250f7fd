package application

import (
	"errors"
	"testing"

	"stagecaption-finalizer/internal/domain"
	"stagecaption-finalizer/internal/store"
	"stagecaption-finalizer/internal/validation"
)

func newTestService(t *testing.T) (*Service, *store.Repository, domain.CaptionProject) {
	t.Helper()
	repo, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc := New(repo, validation.New())
	p, err := svc.CreateProject(CreateProjectCommand{Title: "演出", SourceLanguage: "en", TargetLanguage: "zh", FrameRate: 25, MinDisplayMillis: 500, MaxDisplayMillis: 5000, Actor: "负责人", IdempotencyKey: "create"})
	if err != nil {
		t.Fatal(err)
	}
	return svc, repo, p
}

func currentWorkspace(t *testing.T, svc *Service, projectID string) Workspace {
	t.Helper()
	w, err := svc.GetWorkspace(projectID)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestRuleUpdateInvalidatesReturnedValidationAtomically(t *testing.T) {
	svc, repo, p := newTestService(t)
	r, err := svc.SubmitRevision(p.ID, RevisionCommand{SubmittedBy: "译员", ExpectedVersion: p.Version, IdempotencyKey: "r1", Cues: []domain.CaptionCue{{Sequence: 1, InMillis: 0, OutMillis: 1200, SourceText: "hello", TranslatedText: "你好"}}})
	if err != nil {
		t.Fatal(err)
	}
	w := currentWorkspace(t, svc, p.ID)
	if _, err = svc.Validate(p.ID, VersionedCommand{ExpectedVersion: w.Project.Version, Actor: "译员", IdempotencyKey: "v1"}); err != nil {
		t.Fatal(err)
	}
	w = currentWorkspace(t, svc, p.ID)
	if _, err = svc.SubmitForReview(p.ID, VersionedCommand{ExpectedVersion: w.Project.Version, Actor: "译员", IdempotencyKey: "submit"}); err != nil {
		t.Fatal(err)
	}
	w = currentWorkspace(t, svc, p.ID)
	if _, err = svc.Review(p.ID, ReviewCommand{Decision: "return", Reason: "需按新节奏调整", Reviewer: "校审", ExpectedVersion: w.Project.Version, IdempotencyKey: "return"}); err != nil {
		t.Fatal(err)
	}
	w = currentWorkspace(t, svc, p.ID)
	updated, err := svc.UpdateProjectRules(p.ID, RuleUpdateCommand{Title: "演出", SourceLanguage: "en", TargetLanguage: "zh", FrameRate: 25, MinDisplayMillis: 500, MaxDisplayMillis: 4500, ExpectedVersion: w.Project.Version, Actor: "负责人", IdempotencyKey: "rules"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != w.Project.Version+1 || updated.MaxDisplayMillis != 4500 {
		t.Fatalf("规则未更新: %#v", updated)
	}
	w = currentWorkspace(t, svc, p.ID)
	if w.Validation != nil {
		t.Fatal("规则变化后旧校验仍然有效")
	}
	if _, err = svc.SubmitForReview(p.ID, VersionedCommand{ExpectedVersion: w.Project.Version, Actor: "译员", IdempotencyKey: "stale"}); !errors.Is(err, domain.ErrStaleValidation) {
		t.Fatalf("预期过期校验错误，得到 %v", err)
	}
	beforeEvents := len(repo.Events(p.ID))
	before := w.Project
	_, err = svc.UpdateProjectRules(p.ID, RuleUpdateCommand{Title: "非法", SourceLanguage: "en", TargetLanguage: "zh", FrameRate: 0, MinDisplayMillis: 500, MaxDisplayMillis: 4500, ExpectedVersion: w.Project.Version, Actor: "负责人", IdempotencyKey: "bad-rules"})
	if err == nil {
		t.Fatal("非法规则不应成功")
	}
	after := currentWorkspace(t, svc, p.ID)
	if after.Project != before || len(repo.Events(p.ID)) != beforeEvents {
		t.Fatal("失败规则更新改变了投影或审计")
	}
	if r.ID != after.Project.CurrentRevisionID {
		t.Fatal("当前修订被意外改变")
	}
}

func TestGlossaryBatchConflictAndSnapshotHistory(t *testing.T) {
	svc, repo, p := newTestService(t)
	result, err := svc.ImportGlossary(p.ID, BatchGlossaryCommand{Entries: []GlossaryEntryInput{{SourceText: "King", RequiredTranslation: "王"}, {SourceText: "Queen", RequiredTranslation: "王后"}, {SourceText: "Crown", RequiredTranslation: "王冠"}}, ExpectedVersion: p.Version, Actor: "译员", IdempotencyKey: "batch-1"})
	if err != nil {
		t.Fatal(err)
	}
	if result.ImportedCount != 3 || result.GlossaryVersion != 1 {
		t.Fatalf("批量结果错误: %#v", result)
	}
	for _, term := range result.Entries {
		if term.Version != result.GlossaryVersion {
			t.Fatal("同批术语版本不一致")
		}
	}
	w := currentWorkspace(t, svc, p.ID)
	revision, err := svc.SubmitRevision(p.ID, RevisionCommand{SubmittedBy: "译员", ExpectedVersion: w.Project.Version, IdempotencyKey: "r", Cues: []domain.CaptionCue{{Sequence: 1, InMillis: 0, OutMillis: 1000, SourceText: "King", TranslatedText: "王"}}})
	if err != nil {
		t.Fatal(err)
	}
	w = currentWorkspace(t, svc, p.ID)
	beforeEvents, beforeVersion := len(repo.Events(p.ID)), w.Project.Version
	_, err = svc.ImportGlossary(p.ID, BatchGlossaryCommand{Entries: []GlossaryEntryInput{{SourceText: "Stage", RequiredTranslation: "舞台"}, {SourceText: "Light", RequiredTranslation: "灯光", ForbiddenTranslations: []string{"灯光"}}}, ExpectedVersion: beforeVersion, Actor: "译员", IdempotencyKey: "bad-batch"})
	var conflicts *GlossaryConflictError
	if !errors.As(err, &conflicts) || len(conflicts.Conflicts) != 1 || conflicts.Conflicts[0].Line != 2 {
		t.Fatalf("冲突报告错误: %v %#v", err, conflicts)
	}
	after := currentWorkspace(t, svc, p.ID)
	if after.Project.Version != beforeVersion || len(after.Terms) != 3 || len(repo.Events(p.ID)) != beforeEvents {
		t.Fatal("冲突批次产生部分写入")
	}
	old, err := svc.Glossary(p.ID, revision.GlossaryVersion)
	if err != nil || len(old) != 3 {
		t.Fatalf("旧术语快照不可查询: %v %#v", err, old)
	}
}

func TestRevisionDiffValidationComparisonAndBatchResolve(t *testing.T) {
	svc, _, p := newTestService(t)
	first, err := svc.SubmitRevision(p.ID, RevisionCommand{SubmittedBy: "译员", Summary: "初稿", ExpectedVersion: p.Version, IdempotencyKey: "r1", Cues: []domain.CaptionCue{{Sequence: 1, InMillis: 0, OutMillis: 1000, SourceText: "A", TranslatedText: "甲"}, {Sequence: 2, InMillis: 1100, OutMillis: 1300, SourceText: "B", TranslatedText: "乙"}}})
	if err != nil {
		t.Fatal(err)
	}
	w := currentWorkspace(t, svc, p.ID)
	second, err := svc.SubmitRevision(p.ID, RevisionCommand{SubmittedBy: "译员", Summary: "调整并新增", ExpectedVersion: w.Project.Version, IdempotencyKey: "r2", Cues: []domain.CaptionCue{{Sequence: 1, InMillis: 0, OutMillis: 1000, SourceText: "A", TranslatedText: "甲"}, {Sequence: 2, InMillis: 1100, OutMillis: 1300, SourceText: "B", TranslatedText: "乙改"}, {Sequence: 3, InMillis: 1400, OutMillis: 1500, SourceText: "C", TranslatedText: "丙"}}})
	if err != nil {
		t.Fatal(err)
	}
	diff, err := svc.DiffRevisions(p.ID, first.ID, second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) != 2 || diff.Changes[0].Kind != domain.CueModified || diff.Changes[1].Kind != domain.CueAdded {
		t.Fatalf("修订差异错误: %#v", diff.Changes)
	}
	w = currentWorkspace(t, svc, p.ID)
	run, err := svc.Validate(p.ID, VersionedCommand{ExpectedVersion: w.Project.Version, Actor: "译员", IdempotencyKey: "v1"})
	if err != nil || len(run.Findings) != 2 {
		t.Fatalf("预期两个时长问题: %v %#v", err, run.Findings)
	}
	filtered, err := svc.FilterFindings(p.ID, FindingFilter{RuleCode: "DISPLAY_DURATION"})
	if err != nil || len(filtered) != 2 {
		t.Fatalf("筛选失败: %v %#v", err, filtered)
	}
	w = currentWorkspace(t, svc, p.ID)
	items := []ResolveItem{{FindingID: run.Findings[0].ID, ResolutionNote: "延长显示"}, {FindingID: run.Findings[1].ID, ResolutionNote: "延长显示"}}
	if _, err = svc.ResolveFindings(p.ID, BatchResolveCommand{Items: items, ExpectedVersion: w.Project.Version - 1, Actor: "译员", IdempotencyKey: "stale-resolve"}); !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("预期版本冲突，得到 %v", err)
	}
	stillOpen, _ := svc.FilterFindings(p.ID, FindingFilter{Status: string(domain.FindingOpen)})
	if len(stillOpen) != 2 {
		t.Fatal("过期批量整改产生部分关闭")
	}
	resolved, err := svc.ResolveFindings(p.ID, BatchResolveCommand{Items: items, ExpectedVersion: w.Project.Version, Actor: "译员", IdempotencyKey: "resolve"})
	if err != nil || len(resolved.Resolved) != 2 {
		t.Fatalf("批量整改失败: %v %#v", err, resolved)
	}
	w = currentWorkspace(t, svc, p.ID)
	if _, err = svc.SubmitForReview(p.ID, VersionedCommand{ExpectedVersion: w.Project.Version, Actor: "译员", IdempotencyKey: "no-replacement"}); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("旧修订不应直接送审: %v", err)
	}
	third, err := svc.SubmitRevision(p.ID, RevisionCommand{SubmittedBy: "译员", ExpectedVersion: w.Project.Version, IdempotencyKey: "r3", Cues: []domain.CaptionCue{{Sequence: 1, InMillis: 0, OutMillis: 1000, SourceText: "A", TranslatedText: "甲"}, {Sequence: 2, InMillis: 1100, OutMillis: 1800, SourceText: "B", TranslatedText: "乙改"}, {Sequence: 3, InMillis: 1900, OutMillis: 2500, SourceText: "C", TranslatedText: "丙"}}})
	if err != nil {
		t.Fatal(err)
	}
	w = currentWorkspace(t, svc, p.ID)
	newRun, err := svc.Validate(p.ID, VersionedCommand{ExpectedVersion: w.Project.Version, Actor: "译员", IdempotencyKey: "v2"})
	if err != nil || len(newRun.Findings) != 0 {
		t.Fatalf("替代修订校验失败: %v %#v", err, newRun.Findings)
	}
	comparison, err := svc.CompareValidationRuns(p.ID, run.ID, newRun.ID)
	if err != nil || len(comparison.Disappeared) != 2 || len(comparison.Added) != 0 {
		t.Fatalf("运行对比错误: %v %#v", err, comparison)
	}
	if third.ParentRevisionID != second.ID {
		t.Fatal("替代修订未关联当前父修订")
	}
}

func TestReviewPreviewFreezeAndDeterministicExports(t *testing.T) {
	svc, _, p := newTestService(t)
	r, err := svc.SubmitRevision(p.ID, RevisionCommand{SubmittedBy: "译员", ExpectedVersion: p.Version, IdempotencyKey: "r", Cues: []domain.CaptionCue{{Sequence: 1, InMillis: 0, OutMillis: 1200, SourceText: "Hello", TranslatedText: "你好"}}})
	if err != nil {
		t.Fatal(err)
	}
	w := currentWorkspace(t, svc, p.ID)
	if _, err = svc.Validate(p.ID, VersionedCommand{ExpectedVersion: w.Project.Version, Actor: "译员", IdempotencyKey: "v"}); err != nil {
		t.Fatal(err)
	}
	w = currentWorkspace(t, svc, p.ID)
	if _, err = svc.SubmitForReview(p.ID, VersionedCommand{ExpectedVersion: w.Project.Version, Actor: "译员", IdempotencyKey: "s"}); err != nil {
		t.Fatal(err)
	}
	w = currentWorkspace(t, svc, p.ID)
	if _, err = svc.Review(p.ID, ReviewCommand{Decision: "approve", Reviewer: r.SubmittedBy, ExpectedVersion: w.Project.Version, IdempotencyKey: "self"}); !errors.Is(err, domain.ErrIdentityConflict) {
		t.Fatalf("提交者不应批准: %v", err)
	}
	decision, err := svc.Review(p.ID, ReviewCommand{Decision: "approve", Reviewer: "校审", ExpectedVersion: w.Project.Version, IdempotencyKey: "approve"})
	if err != nil {
		t.Fatal(err)
	}
	detail, err := svc.ReviewDetail(p.ID)
	if err != nil || !detail.InitialRevision || len(detail.Decisions) != 1 {
		t.Fatalf("复核详情错误: %v %#v", err, detail)
	}
	preview, err := svc.FreezePreview(p.ID)
	if err != nil || preview.Review.ID != decision.ID {
		t.Fatalf("冻结预览错误: %v %#v", err, preview)
	}
	w = currentWorkspace(t, svc, p.ID)
	manifest, err := svc.Freeze(p.ID, VersionedCommand{ExpectedVersion: w.Project.Version, Actor: "负责人", IdempotencyKey: "freeze"})
	if err != nil {
		t.Fatal(err)
	}
	if manifest.RuleDigest != preview.RuleDigest || manifest.GlossaryDigest != preview.GlossaryDigest || manifest.CaptionDigest != preview.CaptionDigest {
		t.Fatal("冻结清单与预览摘要不一致")
	}
	replayed, err := svc.Freeze(p.ID, VersionedCommand{ExpectedVersion: 0, Actor: "负责人", IdempotencyKey: "freeze"})
	if err != nil || replayed != manifest {
		t.Fatalf("冻结幂等重放不一致: %v %#v", err, replayed)
	}
	for _, kind := range []string{"captions", "glossary", "audit"} {
		firstExport, exportErr := svc.ExportItem(p.ID, kind)
		if exportErr != nil {
			t.Fatal(exportErr)
		}
		secondExport, exportErr := svc.ExportItem(p.ID, kind)
		if exportErr != nil {
			t.Fatal(exportErr)
		}
		if firstExport.Digest != secondExport.Digest || string(firstExport.Content) != string(secondExport.Content) || firstExport.ManifestID != manifest.ID {
			t.Fatalf("%s 导出不确定", kind)
		}
	}
	verified, err := svc.Verify(p.ID)
	if err != nil || !verified.Valid {
		t.Fatalf("核验失败: %v %#v", err, verified)
	}
}
