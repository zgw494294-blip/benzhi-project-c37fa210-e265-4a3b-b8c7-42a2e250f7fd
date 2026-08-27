package domain

import "fmt"

type DisplayRules struct {
	FrameRate        float64 `json:"frameRate"`
	MinDisplayMillis int64   `json:"minDisplayMillis"`
	MaxDisplayMillis int64   `json:"maxDisplayMillis"`
}

func (p CaptionProject) Rules() DisplayRules {
	return DisplayRules{p.FrameRate, p.MinDisplayMillis, p.MaxDisplayMillis}
}
func (p CaptionProject) ValidateRules() error {
	if p.FrameRate <= 0 {
		return fmt.Errorf("%w: 帧率必须大于零", ErrInvalidInput)
	}
	if p.MinDisplayMillis <= 0 || p.MaxDisplayMillis < p.MinDisplayMillis {
		return fmt.Errorf("%w: 显示时长范围无效", ErrInvalidInput)
	}
	return nil
}
func (p CaptionProject) IsTerminal() bool { return p.Status == StatusFrozen }
func (p CaptionProject) CanSubmitRevision() bool {
	return p.Status == StatusDraft || p.Status == StatusNeedsFix || p.Status == StatusReturned
}
func (p CaptionProject) CanValidate() bool {
	return p.CurrentRevisionID != "" && !p.IsTerminal() && p.Status != StatusInReview && p.Status != StatusApproved
}
func (p CaptionProject) CanReview() bool { return p.Status == StatusInReview }
func (p CaptionProject) CanFreeze() bool { return p.Status == StatusApproved }
