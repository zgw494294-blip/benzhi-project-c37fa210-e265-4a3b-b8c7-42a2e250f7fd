package store

import (
	"stagecaption-finalizer/internal/domain"
	"testing"
)

func TestRecoverAndRejectPartialMutation(t *testing.T) {
	dir := t.TempDir()
	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	err = r.Update("p", "created", "a", func(s *Snapshot) error { s.Projects["p"] = domain.CaptionProject{ID: "p", Version: 1}; return nil })
	if err != nil {
		t.Fatal(err)
	}
	err = r.Update("p", "failed", "a", func(s *Snapshot) error {
		s.Projects["p"] = domain.CaptionProject{ID: "p", Version: 2}
		return domain.ErrInvalidState
	})
	if err == nil {
		t.Fatal("预期失败")
	}
	r2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err = r2.Read(func(s Snapshot) error {
		if s.Projects["p"].Version != 1 {
			t.Fatal("失败操作污染状态")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err = r2.VerifyAudit(); err != nil {
		t.Fatal(err)
	}
}
