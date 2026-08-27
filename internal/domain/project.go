package domain

import (
	"fmt"
	"strings"
	"time"
)

func NewProject(id, title, source, target string, frameRate float64, minMS, maxMS int64, now time.Time) (*CaptionProject, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(title) == "" || strings.TrimSpace(source) == "" || strings.TrimSpace(target) == "" {
		return nil, fmt.Errorf("%w: 项目、标题和语言不能为空", ErrInvalidInput)
	}
	if frameRate <= 0 || minMS <= 0 || maxMS < minMS {
		return nil, fmt.Errorf("%w: 帧率或显示时长范围不合法", ErrInvalidInput)
	}
	return &CaptionProject{ID: id, Title: strings.TrimSpace(title), SourceLanguage: source, TargetLanguage: target, FrameRate: frameRate, MinDisplayMillis: minMS, MaxDisplayMillis: maxMS, Status: StatusDraft, Version: 1, CreatedAt: now.UTC(), UpdatedAt: now.UTC()}, nil
}

func (p *CaptionProject) EnsureMutable() error {
	if p.Status == StatusFrozen {
		return ErrFrozen
	}
	return nil
}

func (p *CaptionProject) CheckVersion(expected int64) error {
	if expected != p.Version {
		return fmt.Errorf("%w: expected=%d actual=%d", ErrVersionConflict, expected, p.Version)
	}
	return nil
}

func (p *CaptionProject) Touch(now time.Time) { p.Version++; p.UpdatedAt = now.UTC() }

func (p *CaptionProject) UpdateRules(title, source, target string, frameRate float64, minMS, maxMS int64, now time.Time) error {
	if p.Status != StatusDraft && p.Status != StatusReturned {
		return ErrInvalidState
	}
	title = strings.TrimSpace(title)
	source = strings.TrimSpace(source)
	target = strings.TrimSpace(target)
	if title == "" || source == "" || target == "" {
		return fmt.Errorf("%w: 标题和语言不能为空", ErrInvalidInput)
	}
	if frameRate <= 0 || minMS <= 0 || minMS > maxMS {
		return fmt.Errorf("%w: 帧率必须为正数且最短时长不得大于最长时长", ErrInvalidInput)
	}
	p.Title = title
	p.SourceLanguage = source
	p.TargetLanguage = target
	p.FrameRate = frameRate
	p.MinDisplayMillis = minMS
	p.MaxDisplayMillis = maxMS
	p.Touch(now)
	return nil
}

func (p *CaptionProject) SetRevision(revisionID string, hasParent bool, now time.Time) error {
	if err := p.EnsureMutable(); err != nil {
		return err
	}
	if p.Status == StatusInReview || p.Status == StatusApproved {
		return ErrInvalidState
	}
	if hasParent && p.CurrentRevisionID == "" {
		return fmt.Errorf("%w: 找不到父修订", ErrInvalidInput)
	}
	p.CurrentRevisionID = revisionID
	p.Status = StatusDraft
	p.Touch(now)
	return nil
}

func (p *CaptionProject) MarkValidated(blocked bool, now time.Time) error {
	if err := p.EnsureMutable(); err != nil {
		return err
	}
	if p.CurrentRevisionID == "" || p.Status == StatusInReview || p.Status == StatusApproved {
		return ErrInvalidState
	}
	if blocked {
		p.Status = StatusNeedsFix
	} else {
		p.Status = StatusValidated
	}
	p.Touch(now)
	return nil
}

func (p *CaptionProject) SubmitReview(now time.Time) error {
	if p.Status != StatusValidated {
		return ErrInvalidState
	}
	p.Status = StatusInReview
	p.Touch(now)
	return nil
}

func (p *CaptionProject) ApplyReview(approved bool, now time.Time) error {
	if p.Status != StatusInReview {
		return ErrInvalidState
	}
	if approved {
		p.Status = StatusApproved
	} else {
		p.Status = StatusReturned
	}
	p.Touch(now)
	return nil
}

func (p *CaptionProject) Freeze(now time.Time) error {
	if p.Status != StatusApproved {
		return ErrInvalidState
	}
	p.Status = StatusFrozen
	p.Touch(now)
	return nil
}
